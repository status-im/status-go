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
    lmn = {
      url = "git+https://github.com/logos-messaging/logos-messaging-nim?submodules=1&rev=cccc8ab6fda0e54752936db0d5c80b02a2c34a3a";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    # We cannot do follows since the nim-unwrapped-2_0 doesn't exist in this nixpkgs version above
    nim-sds.url = "git+https://github.com/logos-messaging/nim-sds?ref=refs/heads/start-using-nimble&rev=1c904d7d8840bd03233db6e02dceb86b263d7bdb";
  };

  outputs = { self, nixpkgs, lmn, nim-sds }:
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
            libwaku        = lmn.packages.${system}.libwaku;
            lib-sds-pkg  = nim-sds.packages.${system}.libsds;
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
