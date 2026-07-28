{ pkgs }:

let
  inherit (pkgs) lib stdenv;
  inherit (stdenv.hostPlatform) isDarwin isWindows isCygwin isAarch64;

in pkgs.mkShell {
  name = "status-go-shell";

  buildInputs = with pkgs; [
    git jq which gcc rustc cargo openjdk openssl nim go
    golangci-lint go-junit-report gopls
    protobuf_29 protoc-gen-go gotestsum
    libsds libwaku libstorage
  ] ++ lib.optionals (isDarwin) [
    pkgs.xcodeWrapper
  ] ++ lib.optionals (!(stdenv.isLinux && isAarch64)) [
    codecov-cli
  ];

  shellHook = ''
    export USE_SYSTEM_NIM=1

    export LIBWAKU="$(echo ${pkgs.libwaku}/lib/libwaku.*)"
    export LIBSDS="$(echo ${pkgs.libsds}/lib/libsds.*)"
    export LIBSTORAGE="$(echo ${pkgs.libstorage}/lib/libstorage.*)"
    export LOGOS_STORAGE_LIB_DIR="${pkgs.libstorage}/lib"
    export LOGOS_STORAGE_INC_DIR="${pkgs.libstorage}/include"
    export NIM_SDS_INC_DIR="${pkgs.libsds}/include"
    export NIM_SDS_LIB_DIR="${pkgs.libsds}/lib"

    export LD_LIBRARY_PATH="${lib.makeLibraryPath (with pkgs; [libwaku libsds libstorage])}"
  '' + lib.optionalString (!isDarwin && isAarch64) ''
    export ANDROID_HOME=${pkgs.androidPkgs.androidsdk}/libexec/android-sdk/
    export ANDROID_NDK=\$ANDROID_HOME/ndk-bundle
    export ANDROID_SDK_ROOT=\$ANDROID_HOME
    export ANDROID_NDK_HOME=\$ANDROID_NDK
  '' + lib.optionalString (stdenv.isDarwin) ''
    export PATH="/usr/bin:$PATH"
  '';

  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = stdenv.isDarwin;
}
