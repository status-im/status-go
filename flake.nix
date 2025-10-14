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
    nwaku.url = "git+https://github.com/waku-org/nwaku?submodules=1&rev=e755fd834f5f3d6fba216b09469316f0328b3b6f";
  };

  outputs = { self, nixpkgs, nwaku }:
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

            # 🧩 Add nim-sds dependency here (fetched once, no git clone)
            nim-sds = prev.fetchFromGitHub {
              owner = "waku-org";
              repo = "nim-sds";
              rev = "23d001adb94436d886d66258a11ae19669ac8f71";
              sha256 = "sha256-2z/3VTWkN3UW3NdX6S+QK0s4sCdEqbRJRkKJQig7fJc=";
            };
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

          # Patch sds Makefile to avoid network clone
          patchPhase = ''
            echo "Patching sds Makefile to avoid network clone..."
            substituteInPlace vendor/github.com/waku-org/sds-go-bindings/sds/Makefile \
              --replace-warn "git clone https://github.com/waku-org/nim-sds" \
                        "cp -r ${pkgs.nim-sds} nim-sds"
          '';

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
