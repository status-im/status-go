{
  description = "Status Go Flake";

  nixConfig = {
    extra-substituters = [ "https://nix-cache.status.im/" ];
    extra-trusted-public-keys = [ "nix-cache.status.im-1:x/93lOfLU+duPplwMSBR+OlY4+mo+dCN7n0mr4oPwgY=" ];
    # Some downloads are multiple GB, default is 5 minutes
    stalled-download-timeout = 3600;
    connect-timeout = 10;
    max-jobs = "auto";
    # Some builds on MacOS have issue with sandbox so they are disabled with __noChroot.
    sandbox = "relaxed";
  };

  inputs = {
    # We are pinning the commit because ultimately we want to use same commit across different projects.
    # A commit from nixpkgs 24.11 release : https://github.com/NixOS/nixpkgs/tree/release-24.11
    nixpkgs.url = "github:NixOS/nixpkgs/0ef228213045d2cdb5a169a95d63ded38670b293";
    # We cannot do follows since the nim-unwrapped-2_0 doesn't exist in this nixpkgs version above
    nwaku.url = "git+https://github.com/waku-org/nwaku?submodules=1&rev=7e5041d5e17d717f77fb74ffd876b987a0c6bf5d";
    nim-sds.url = "git+https://github.com/waku-org/nim-sds?submodules=1&rev=f3b084103dea467657f737c0e6b6a63db10e097c";
  };

  outputs = { self, nixpkgs, nwaku, nim-sds }:
  let
    stableSystems = [
      "x86_64-linux" "aarch64-linux"
      "x86_64-darwin" "aarch64-darwin"
      "x86_64-windows"
    ];
    forAllSystems = f: nixpkgs.lib.genAttrs stableSystems (system: f system);
    pkgsOverlay = import ./nix/overlay.nix;
    pkgsFor = forAllSystems (
      system: import nixpkgs {
        inherit system;
        config = {
          android_sdk.accept_license = true;
          allowUnfree = true;
        };
        overlays = [
          pkgsOverlay
          (final: prev: let
            libSds = nim-sds.packages.${system}.libsds;
          in {
            # Make nwaku available
            nwaku = nwaku.packages.${system};

            # Wrap nim-sds package so its dependencies propagate
            lib-sds-pkg = libSds.overrideAttrs (old: {
              propagatedBuildInputs = with final; [
                openssl
                gmp
                nim-unwrapped-2_2
              ];
            });
          })
        ];
      }
    );
  in {
    devShells = forAllSystems (system: {
      default = pkgsFor.${system}.callPackage ./nix/shell.nix { };
    });

    packages = forAllSystems (system: let
      pkgs = pkgsFor.${system};
      statusGo = import ./nix/pkgs/status-go { inherit self pkgs; };
    in {
      status-go-library = statusGo.library;
      status-go-mobile-android = statusGo.mobile.android {};
      status-go-mobile-ios = statusGo.mobile.ios {};
    });
  };
}
