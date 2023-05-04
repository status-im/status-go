{
  /* This should match Nixpkgs commit in status-mobile. */
  source ? builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/8fd4097529b4a71e0de000c90c8c4279f534eada.zip";
    sha256 = "sha256:174p05mzfqk56l4lqvwkbqka4qi5imbd75xaaw4bzmr00bails9f";
  },
  pkgs ? import (source){
    config = {
      allowUnfree = true;
      android_sdk.accept_license = true;
    };
    overlays = [
      (self: super: {
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
    go_1_19 golangci-lint gopls go-bindata gomobileMod
    mockgen protobuf3_17 protoc-gen-go
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
