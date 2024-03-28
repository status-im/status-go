# Override some packages and utilities in 'pkgs'
# and make them available globally via callPackage.
#
# For more details see:
# - https://nixos.wiki/wiki/Overlays
# - https://nixos.org/nixos/nix-pills/callpackage-design-pattern.html
final: prev:
let
  inherit (prev) callPackage;
in {
  androidPkgs = prev.androidenv.composeAndroidPackages {
    toolsVersion = "26.1.1";
    platformToolsVersion = "33.0.3";
    buildToolsVersions = [ "31.0.0" ];
    platformVersions = [ "31" ];
    cmakeVersions = [ "3.18.1" ];
    ndkVersion = "22.1.7171670";
    includeNDK = true;
    includeExtras = [
      "extras;android;m2repository"
      "extras;google;m2repository"
    ];
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
}
