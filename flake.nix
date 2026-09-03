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
    # A commit from nixpkgs 25.11 release : https://github.com/NixOS/nixpkgs/tree/release-25.11
    nixpkgs.url = "github:NixOS/nixpkgs/535f3e6942cb1cead3929c604320d3db54b542b9";
    logos-storage-nim = {
      # TODO: temporary pin, see https://github.com/logos-storage/logos-storage-nim/pull/1496
      url = "git+https://github.com/logos-storage/logos-storage-nim?submodules=1&rev=978c560aed6dc4cff6b602e94fc0fcc66fea6399";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    # We cannot do follows since the nim-unwrapped-2_0 doesn't exist in this nixpkgs version above
    nim-sds.url = "git+https://github.com/logos-messaging/nim-sds?submodules=1&rev=04441cba115e59c5f3ad959ea83ddffee68037cb";
    # nimble 0.20.1 in the nixpkgs above cannot resolve nim-sds through the
    # bindings; 0.24.1 is what logos-delivery builds with too.
    nixpkgs-nimble.url = "github:NixOS/nixpkgs/34ab99075ac4f7e40cf037eef32cb1c360bb85e9";
  };

  outputs = { self, nixpkgs, logos-storage-nim, nim-sds, nixpkgs-nimble }:
  let
    stableSystems = [
      "x86_64-linux" "aarch64-linux"
      "x86_64-darwin" "aarch64-darwin"
      "x86_64-windows"
    ];
    forAllSystems = f: nixpkgs.lib.genAttrs stableSystems (system: f system);
    pkgsOverlay = import ./nix/overlay.nix;
    # nim-sds/logos-storage-nim hardcode XDG_CACHE_HOME=/tmp, which causes
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
            libsds     = useTmpdirForNimCache nim-sds.packages.${system}.libsds;
            libstorage = useTmpdirForNimCache logos-storage-nim.packages.${system}.libstorage;
            nimble = nixpkgs-nimble.legacyPackages.${system}.nimble;
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
