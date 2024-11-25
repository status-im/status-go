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

  go = prev.go_1_21;
  buildGoModule = prev.buildGo121Module;
  buildGoPackage = prev.buildGo121Package;

  golangci-lint = prev.golangci-lint.override {
    buildGoModule = args: prev.buildGo121Module ( args // rec {
      version = "1.54.0";
      src = prev.fetchFromGitHub {
        owner = "golangci";
        repo = "golangci-lint";
        rev = "v${version}";
        hash = "sha256-UXN5gN1SNv3uvBCliJQ+5PSGHRL7RyU6pmZtGUTFsrQ=";
      };
      vendorHash = "sha256-jUlK/A0HxBrIby2C0zYFtnxQX1bgKVyypI3QdH4u/rg=";
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

  # Custom packages
  go-modvendor = callPackage ./pkgs/go-modvendor { };
  codecov-cli = callPackage ./pkgs/codecov-cli { };
  go-generate-fast = callPackage ./pkgs/go-generate-fast { };

  gomobile = (prev.gomobile.overrideAttrs (old: {
    patches = [
      (final.fetchurl { # https://github.com/golang/mobile/pull/84
        url = "https://github.com/golang/mobile/commit/f20e966e05b8f7e06bed500fa0da81cf6ebca307.patch";
        sha256 = "sha256-TZ/Yhe8gMRQUZFAs9G5/cf2b9QGtTHRSObBFD5Pbh7Y=";
      })
      (final.fetchurl { # https://github.com/golang/go/issues/58426
        url = "https://github.com/golang/mobile/commit/406ed3a7b8e44dc32844953647b49696d8847d51.patch";
        sha256 = "sha256-dqbYukHkQEw8npOkKykOAzMC3ot/Y4DEuh7fE+ptlr8=";
      })
      (final.fetchurl { # https://github.com/golang/go/issues/63141
        url = "https://github.com/golang/mobile/commit/e2f452493d570cfe278e63eccec99e62d4c775e5.patch";
        sha256 = "sha256-gFcy/Ikh7MzmDx5Tpxe3qCnP36+ZTKU2XkJGH6n5l7Q=";
      })
    ];
  }));
}
