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
    platformToolsVersion = "35.0.2";
    buildToolsVersions = [ "34.0.0" ];
    platformVersions = [ "34" ];
    cmakeVersions = [ "3.22.1" ];
    ndkVersion = "27.2.12479018";
    includeNDK = true;
    includeExtras = [
      "extras;android;m2repository"
      "extras;google;m2repository"
    ];
  };

  openjdk = prev.openjdk17_headless;

  go = prev.go_1_26;
  buildGoModule = prev.buildGo126Module;

  golangci-lint = prev.golangci-lint.overrideAttrs (oldAttrs: rec {
    version = "2.12.2";
    src = prev.fetchFromGitHub {
      owner = "golangci";
      repo = "golangci-lint";
      rev = "v${version}";
      sha256 = "sha256-qR7fp1x2S+EwEAcplRHTvA3jWwLr/XSiYKSZtAwkrNU=";
    };
    vendorHash = "sha256-AG5wtLwWLz55bdp1oi3cW+9O3yj1W1P7MV9zxym7Pb4=";
  });

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
  codecov-cli = callPackage ./pkgs/codecov-cli { };

  # Not yet packaged in the pinned nixpkgs.
  protobuf_36 = callPackage "${prev.path}/pkgs/development/libraries/protobuf/generic.nix" {
    version = "36.0";
    hash = "sha256-VGXFfqLm7IEJ9MQpMYhdVW5qPZbrYZ6q+0Y1TqQkjks=";
  };
}
