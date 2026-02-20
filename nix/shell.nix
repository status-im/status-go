{ pkgs }:

let
  inherit (pkgs) lib stdenv;
  inherit (stdenv.hostPlatform) isDarwin isWindows isCygwin isAarch64;
  /* Override the default SDK to enable darwin-x86_64 builds */
  appleSdk11Stdenv = pkgs.overrideSDK stdenv "11.0";
  sdk11mkShell = pkgs.mkShell.override { stdenv = appleSdk11Stdenv; };
  mkShell = if isDarwin then sdk11mkShell else pkgs.mkShell;

in mkShell {
  name = "status-go-shell";

  buildInputs = with pkgs; [
    git jq which gcc rustc cargo openjdk openssl nim go
    golangci-lint go-junit-report gopls codecov-cli
    protobuf3_24 protoc-gen-go gotestsum
    libsds libwaku libstorage
  ] ++ lib.optionals (isDarwin) [
    pkgs.xcodeWrapper
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
    if [ -f "${pkgs.libstorage}/lib/libstorage.so" ]; then
      storage_revision_line=$(${pkgs.binutils}/bin/strings "${pkgs.libstorage}/lib/libstorage.so" | ${pkgs.ripgrep}/bin/rg -m1 "Storage revision:" || true)
      if [ -n "$storage_revision_line" ]; then
        echo "libstorage revision: $storage_revision_line"
      fi
    fi
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
