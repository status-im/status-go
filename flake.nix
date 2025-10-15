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
    nim-sds.url = "git+https://github.com/waku-org/nim-sds?submodules=1&rev=b74622da64826415dd87186f5c3caf1f6cc29646";
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
          (final: prev: {
            # Make nwaku available
            nwaku = nwaku.packages.${system};

            # Make nim-sds available
            nim-sds = nim-sds.packages.${system};
          })
        ];
      }
    );
  in {
    devShells = forAllSystems (system: {
      default = pkgsFor.${system}.callPackage ./nix/shell.nix { };
    });

    packages = forAllSystems (system:
      let
        pkgs = pkgsFor.${system};

        # Import your Status Go packaging logic
        statusGo = import ./nix/pkgs/status-go { inherit self pkgs; };
      in
      let
        statusGoLibrary = pkgs.stdenv.mkDerivation {
          inherit (statusGo.library) pname version src nativeBuildInputs;

          # Add Go, git, and GNU Make
          buildInputs = statusGo.library.buildInputs ++ [ pkgs.go pkgs.git pkgs.gnumake pkgs.protobuf ];

          # Reuse buildPhase etc. if needed
          inherit (statusGo.library) preBuild buildPhase installPhase;
        };
      in {
        status-go-library = statusGoLibrary;
        status-go-mobile-android = statusGo.mobile.android { };
        status-go-mobile-ios = statusGo.mobile.ios { };
      });
  };
}
