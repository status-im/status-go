{
  /* This should match Nixpkgs commit in status-mobile. */
  source ? builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/ddf0003c57fb5cbb4a9754f2f6d5ebe9cdae5151.tar.gz";
    sha256 = "sha256:07dl6m20119qqchjkf2m47zw1ih2r21zl9d073a0w9n4kcl6n0v7";
  },
  pkgs ? import (source){
    config = {
      allowUnfree = true;
      android_sdk.accept_license = true;
    };
    overlays = [
      (final: prev: {
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
        # https://github.com/golang/go/issues/58426
        gomobile = prev.gomobile.override {
          buildGoModule = args: prev.buildGo120Module ( args // rec {
            version = "unstable-2023-11-27";
            src = prev.fetchgit {
              rev = "76ac6878050a2eef81867f2c6c21108e59919e8f";
              name = "gomobile";
              url = "https://go.googlesource.com/mobile";
              sha256 = "sha256-mq7gKccvI7VCBEiQTueWxMPOCgg/MGE8y2+BlwWx5pw=";
            };
            vendorHash = "sha256-8OBLVd4zs89hoJXzC8BPRgrYjjR7DiA39+7tTaSYUFI=";
          });
        };
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
    go_1_21 golangci-lint go-junit-report gopls go-bindata gomobileMod
    mockgen protobuf3_20 protoc-gen-go
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
