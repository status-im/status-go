# Override some packages and utilities in 'pkgs'
# and make them available globally via callPackage.
#
# For more details see:
# - https://nixos.wiki/wiki/Overlays
# - https://nixos.org/nixos/nix-pills/callpackage-design-pattern.html
final: prev:
let
  inherit (prev) callPackage;
in rec {
  androidPkgs = prev.androidenv.composeAndroidPackages {
    cmdLineToolsVersion = "9.0";
    toolsVersion = "26.1.1";
    platformToolsVersion = "33.0.3";
    buildToolsVersions = [ "34.0.0" ];
    platformVersions = [ "34" ];
    cmakeVersions = [ "3.22.1" ];
    ndkVersion = "25.2.9519653";
    includeNDK = true;
    includeExtras = [
      "extras;android;m2repository"
      "extras;google;m2repository"
    ];
  };

  openjdk = prev.openjdk17_headless;

  go = prev.go_1_22;
  buildGoModule = prev.buildGo122Module;
  buildGoPackage = prev.buildGo122Package;

  golangci-lint = prev.golangci-lint.override {
    buildGoModule = args: prev.buildGo122Module ( args // rec {
      version = "1.59.1";
      src = prev.fetchFromGitHub {
        owner = "golangci";
        repo = "golangci-lint";
        rev = "v${version}";
        hash = "sha256-VFU/qGyKBMYr0wtHXyaMjS5fXKAHWe99wDZuSyH8opg=";
      };
      vendorHash = "sha256-yYwYISK1wM/mSlAcDSIwYRo8sRWgw2u+SsvgjH+Z/7M=";
    });
  };

  go-junit-report = prev.go-junit-report.overrideAttrs ( attrs : rec {
    version = "2.1.0";
    src = prev.fetchFromGitHub {
     owner = "jstemmer";
     repo = "go-junit-report";
     rev = "v${version}";
     sha256 = "sha256-s4XVjACmpd10C5k+P3vtcS/aWxI6UkSUPyxzLhD2vRI=";
    };
  });

  xcodeWrapper = callPackage ./pkgs/xcodeenv/compose-xcodewrapper.nix { } {
    versions = ["16.0" "16.1" "16.2"];
  };

  # Custom packages
  go-modvendor = callPackage ./pkgs/go-modvendor { };
  codecov-cli = callPackage ./pkgs/codecov-cli { };
  go-generate-fast = callPackage ./pkgs/go-generate-fast { };
  # brough in gomobile derivation from https://github.com/NixOS/nixpkgs/commit/f5abef98e8b8c9f9e6da4bdab63f8be1e57ea8c0
  # enabled CGO_ENABLED for status-mobile
  # swapped --replace plag with --replace-queit in substituteInPlace block because its deprecated in newer nix versions
  # swapped buildGoModule with buildGo122Module to ensure derivation is built with go 1.22
  gomobile = callPackage ./pkgs/gomobile {};
}
