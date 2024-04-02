{
  /* This should match Nixpkgs commit in status-mobile. */
  source ? builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/224fd9a362487ab2894dac0df161c84ab1d8880b.tar.gz";
    sha256 = "sha256:1syvl39pi1h8lf5gkd9h7ksn5hp34cj7pa3abr59217kv0bdklhy";
  },
  pkgs ? import (source){
    config = {
      allowUnfree = true;
      android_sdk.accept_license = true;
    };
    overlays = [
      (final: prev: {
        androidPkgs = pkgs.androidenv.composeAndroidPackages {
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
      })
    ];
  }
}:

let
  inherit (pkgs) lib stdenv;

  /* No Android SDK for Darwin aarch64. */
  isMacM1 = stdenv.isDarwin && stdenv.isAarch64;
  /* Lock requires Xcode verison. */
  xcodeWrapper = pkgs.xcodeenv.composeXcodeWrapper {
    version = "14.3";
    allowHigher = true;
  };
  /* Gomobile also needs the Xcode wrapper. */
  gomobileMod = pkgs.gomobile.override {
    inherit xcodeWrapper;
    withAndroidPkgs = !isMacM1;
  };
in pkgs.mkShell {
  name = "status-go-shell";

  buildInputs = with pkgs; [
    git jq which
    go_1_20 golangci-lint go-junit-report gopls go-bindata gomobileMod
    mockgen protobuf3_20 protoc-gen-go gotestsum
  ] ++ lib.optional stdenv.isDarwin xcodeWrapper;

  shellHook = lib.optionalString (!isMacM1) ''
    ANDROID_HOME=${pkgs.androidPkgs.androidsdk}/libexec/android-sdk
    ANDROID_NDK=$ANDROID_HOME/ndk-bundle
    ANDROID_SDK_ROOT=$ANDROID_HOME
    ANDROID_NDK_HOME=$ANDROID_NDK
  '';

  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = stdenv.isDarwin;
}
