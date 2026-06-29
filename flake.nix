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
      # https://github.com/vacp2p/zerokit/commit/b1e4e485ad8e7a13b402c0ad2604ef879d927be4
      inputs.zerokit.url = "github:vacp2p/zerokit/b1e4e485ad8e7a13b402c0ad2604ef879d927be4";
    };
    logos-storage-nim = {
      url = "git+https://github.com/logos-storage/logos-storage-nim?submodules=1&rev=3c09f008bb5266a669fd19f18368f9e8b861b664";
      inputs.nixpkgs.follows = "nixpkgs";
      # https://github.com/logos-storage/circom-compat-ffi/pull/11
      inputs.circom-compat.url = "github:logos-storage/circom-compat-ffi/3cca4e91054a4d2bb5335feb19138362ce0fe417";
    };
    # We cannot do follows since the nim-unwrapped-2_0 doesn't exist in this nixpkgs version above
    nim-sds.url = "git+https://github.com/logos-messaging/nim-sds?submodules=1&rev=d9d70cc2fe7d82bd981b745dbc986353dfc626b9";
  };

  outputs = { self, nixpkgs, lmn, logos-storage-nim, nim-sds }:
  let
    stableSystems = [
      "x86_64-linux" "aarch64-linux"
      "x86_64-darwin" "aarch64-darwin"
      "x86_64-windows"
    ];
    forAllSystems = f: nixpkgs.lib.genAttrs stableSystems (system: f system);
    pkgsOverlay = import ./nix/overlay.nix;
    # nim-sds/lmn/logos-storage-nim hardcode XDG_CACHE_HOME=/tmp, which causes
    # nim cache collisions at /tmp/nim/<name>_d/ on macOS
    useTmpdirForNimCache = drv: drv.overrideAttrs (old: {
      preBuild = ''export XDG_CACHE_HOME="$TMPDIR"'' + "\n" + (old.preBuild or "");
    });
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
            libwaku    = useTmpdirForNimCache lmn.packages.${system}.libwaku;
            libsds     = useTmpdirForNimCache nim-sds.packages.${system}.libsds;
            libstorage = useTmpdirForNimCache logos-storage-nim.packages.${system}.libstorage;
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
