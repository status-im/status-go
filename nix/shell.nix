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

  buildInputs = with pkgs; [
    git jq which
    go golangci-lint go-junit-report gopls codecov-cli
    protobuf3_24 protoc-gen-go gotestsum openjdk openssl
    rustc cargo nim
    lib-sds-pkg libwaku
  ] ++ lib.optionals (stdenv.isDarwin) [
    xcodeWrapper
  ];

  shellHook = ''
    export USE_SYSTEM_NIM=1

    export LIBWAKU_PATH="${pkgs.libwaku}"
    export LIBSDS_PATH="${pkgs.lib-sds-pkg}"
    export LIBWAKU="$(echo ${pkgs.libwaku}/lib/libwaku.*)"
    export LIBSDS="$(echo ${pkgs.lib-sds-pkg}/lib/libsds.*)"
    export NIM_SDS_INC_DIR="${pkgs.lib-sds-pkg}/include"
    export NIM_SDS_LIB_DIR="${pkgs.lib-sds-pkg}/lib"

  ''
  + lib.optionalString (!isMacM1) ''
    export ANDROID_HOME=${pkgs.androidPkgs.androidsdk}/libexec/android-sdk/
    export ANDROID_NDK=\$ANDROID_HOME/ndk-bundle
    export ANDROID_SDK_ROOT=\$ANDROID_HOME
    export ANDROID_NDK_HOME=\$ANDROID_NDK
  ''
  + lib.optionalString (stdenv.isDarwin) ''
    export PATH="/usr/bin:$PATH"
  '';
  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = stdenv.isDarwin;
}
