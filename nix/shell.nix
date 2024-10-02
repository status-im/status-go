{ config ? {}
, pkgs ? import ./pkgs.nix { inherit config; } }:

let
  inherit (pkgs) lib stdenv callPackage;
  /* No Android SDK for Darwin aarch64. */
  isMacM1 = stdenv.isDarwin && stdenv.isAarch64;

  /* Lock requires Xcode verison. */
  xcodeWrapper = callPackage ./pkgs/xcodeenv/compose-xcodewrapper.nix { } {
      versions = ["14.3" "15.1" "15.2" "15.3" "15.4"];
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
    go golangci-lint go-junit-report gopls go-bindata gomobileMod openssl
    mockgen protobuf3_20 protoc-gen-go gotestsum go-modvendor openjdk cc-test-reporter
   ] ++ lib.optionals (stdenv.isDarwin) [ xcodeWrapper ];

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

