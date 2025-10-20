{
  pkgs,
}:

let
  inherit (pkgs) lib stdenv callPackage;
  /* No Android SDK for Darwin aarch64. */
  isMacM1 = stdenv.isDarwin && stdenv.isAarch64;

  /* Override the default SDK to enable darwin-x86_64 builds */
  appleSdk11Stdenv = pkgs.overrideSDK pkgs.stdenv "11.0";
  sdk11mkShell = pkgs.mkShell.override { stdenv = appleSdk11Stdenv; };
  mkShell = if stdenv.isDarwin then sdk11mkShell else pkgs.mkShell;

in mkShell {
  name = "status-go-shell";

  buildInputs = with pkgs;
    lib.optionals (stdenv.isDarwin) [ xcodeWrapper ] ++ [
    git jq which
    go golangci-lint go-junit-report gopls codecov-cli go-generate-fast
    protobuf3_24 protoc-gen-go gotestsum openjdk openssl
    rustc cargo
  ];

  shellHook = lib.optionalString (!isMacM1) ''
    export LD_LIBRARY_PATH=\$LD_LIBRARY_PATH:${pkgs.lib-sds-pkg}/lib/
    ANDROID_HOME=${pkgs.androidPkgs.androidsdk}/libexec/android-sdk/
    ANDROID_NDK=$ANDROID_HOME/ndk-bundle
    ANDROID_SDK_ROOT=$ANDROID_HOME
    ANDROID_NDK_HOME=$ANDROID_NDK
  '' + lib.optionalString (stdenv.isDarwin) ''
    export PATH="/usr/bin:$PATH"
  '';
  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = stdenv.isDarwin;
}

