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
    echo "Patching env.sh to use Nix Nim..."
    env_sh="/go/src/github.com/status-im/status-go/vendor/github.com/waku-org/sds-go-bindings/third_party/nim-sds/vendor/nimbus-build-system/scripts/env.sh"
    if [ -f "$env_sh" ]; then
      echo "Replacing nim by ${pkgs.nim}/bin/nim"
      ${pkgs.gnused}/bin/sed -i 's#/vendor/Nim/bin/nim#${pkgs.nim}/bin/nim#g' "$env_sh"
    else
      echo "Warning: $env_sh not found (PWD=$PWD)"
    fi

    export NIM_SDS_REPO_PATH=${pkgs.nim-sds-src}
    export LD_LIBRARY_PATH=\$LD_LIBRARY_PATH:${pkgs.lib-sds-pkg}/lib/
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

