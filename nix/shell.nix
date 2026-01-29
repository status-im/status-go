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
    gcc go golangci-lint go-junit-report gopls codecov-cli
    protobuf3_24 protoc-gen-go gotestsum openjdk openssl
    rustc cargo
    nim
    lib-sds-pkg
    libwaku
    libstorage
  ];

  shellHook = ''
    export USE_SYSTEM_NIM=1

    export LIBWAKU_PATH="${pkgs.libwaku}"
    export LIBSTORAGE_PATH="${pkgs.libstorage}"
    export LIBSDS_PATH="${pkgs.lib-sds-pkg}"

    export LD_LIBRARY_PATH="${pkgs.libstorage}/lib:${pkgs.libwaku}/bin:${pkgs.lib-sds-pkg}/lib:''${LD_LIBRARY_PATH:-}"

    export CGO_CFLAGS="-I${pkgs.libstorage}/include -I${pkgs.libwaku}/include -I${pkgs.lib-sds-pkg}/include"
    export CGO_LDFLAGS="-L${pkgs.libstorage}/lib -lstorage -Wl,-rpath,${pkgs.libstorage}/lib -L${pkgs.libwaku}/bin -L${pkgs.lib-sds-pkg}/lib -lsds"

    echo "CGO_CFLAGS: $CGO_CFLAGS"
    echo "CGO_LDFLAGS: $CGO_LDFLAGS"
    if [ -f "${pkgs.libstorage}/lib/libstorage.so" ]; then
      storage_revision_line=$(${pkgs.binutils}/bin/strings "${pkgs.libstorage}/lib/libstorage.so" | ${pkgs.ripgrep}/bin/rg -m1 "Storage revision:" || true)
      if [ -n "$storage_revision_line" ]; then
        echo "libstorage revision: $storage_revision_line"
      fi
    fi
  ''
  + lib.optionalString (!isMacM1) ''
    export ANDROID_HOME=${pkgs.androidPkgs.androidsdk}/libexec/android-sdk/
    export ANDROID_NDK=\$ANDROID_HOME/ndk-bundle
    export ANDROID_SDK_ROOT=\$ANDROID_HOME
    export ANDROID_NDK_HOME=\$ANDROID_NDK
  ''
  + lib.optionalString (stdenv.isDarwin) ''
    export PATH="/usr/bin:$PATH"
    export DYLD_LIBRARY_PATH="${pkgs.libstorage}/lib:${pkgs.libwaku}/bin:${pkgs.lib-sds-pkg}/lib:''${DYLD_LIBRARY_PATH:-}"
  ''
  + lib.optionalString (!stdenv.isDarwin) ''
    export LD_LIBRARY_PATH="${pkgs.libstorage}/lib:${pkgs.libwaku}/bin:${pkgs.lib-sds-pkg}/lib:''${LD_LIBRARY_PATH:-}"
  '';
  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = stdenv.isDarwin;
}
