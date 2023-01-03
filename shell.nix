{
  /* This should match Nixpkgs commit in status-mobile. */
  source ? builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/579238da5f431b7833a9f0681663900aaf0dd1e8.zip";
    sha256 = "sha256:0a77c8fq4145k0zdmsda9cmhfw84ipf9nhvvn0givzhza1500g3h";
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
          platformToolsVersion = "33.0.2";
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

pkgs.mkShell {
  name = "status-go-shell";

  buildInputs = with pkgs; [
    git jq which
    go_1_18 gopls go-bindata
    mockgen protobuf3_17 protoc-gen-go
    (gomobile.override { xcodeWrapperArgs = { version = "13.4.1"; }; })
  ] ++ pkgs.lib.optionals pkgs.stdenv.isDarwin [
    (xcodeenv.composeXcodeWrapper { version = "13.4.1"; })
  ];

  shellHook = let
    androidSdk = pkgs.androidPkgs.androidsdk;
  in ''
    ANDROID_HOME=${androidSdk}/libexec/android-sdk
    ANDROID_SDK_ROOT=$ANDROID_HOME
    ANDROID_NDK=${androidSdk}/libexec/android-sdk/ndk-bundle
    ANDROID_NDK_HOME=$ANDROID_NDK
  '';

  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = pkgs.stdenv.isDarwin;
}
