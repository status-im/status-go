{ pkgs,
  self,
  meta,
  version,
  platform ? "android",
  platformVersion ? "23",
  targets ? [ "android/arm64" ],
  goBuildFlags ? [ ], # Use -v or -x for debugging.
  goBuildLdFlags ? [ ],
  outputFileName,
}:

let
  inherit (pkgs.lib) concatStringsSep optionalString optional;
  isIOS = platform == "ios";
  isAndroid = platform == "android";
  goPackagePath = "github.com/status-im/status-go";
in pkgs.buildGoPackage rec {
  pname = "status-go";
  src = builtins.path { path = ./../../../..; name = "status-go-mobile-${platform}"; };

  inherit meta version goPackagePath;

  # Sandbox causes Xcode issues on MacOS. Requires sandbox=relaxed.
  # https://github.com/status-im/status-mobile/pull/13912
  __noChroot = isIOS;

  extraSrcPaths = [ pkgs.gomobile ];

  nativeBuildInputs = let
    # Fixes fatal: not a git repository (or any of the parent directories): .git
    fakeGit = pkgs.writeScriptBin "git" "echo ${version}";
  in
    with pkgs; [
      gomobile
      removeReferencesTo
      go-bindata
      mockgen
      protoc-gen-go
      protobuf3_20
      fakeGit
  ] ++ optional isAndroid pkgs.openjdk_headless
    ++ optional isAndroid pkgs.nwaku.libwaku-android-arm64
    ++ optional isIOS pkgs.xcodeWrapper;

  ldflags = goBuildLdFlags;

  ANDROID_HOME = optionalString isAndroid "${pkgs.androidPkgs.androidsdk}/libexec/android-sdk/";

  # https://pkg.go.dev/net#hdr-Name_Resolution
  # https://github.com/status-im/status-mobile/issues/19736
  # https://github.com/status-im/status-mobile/issues/19581
  # TODO: try removing when go is upgraded to 1.22
  GODEBUG = "netdns=cgo+2";

  # Sentry for status-go
  SENTRY_CONTEXT_NAME = "status-mobile";
  SENTRY_CONTEXT_VERSION = version;

  preBuild = ''
    echo 'Generate static files'
    pushd go/src/$goPackagePath
    make generate SHELL=$SHELL GO111MODULE=on GO_GENERATE_CMD='go generate'
    popd
  '';

  buildPhase = ''
    runHook preBuild
    echo -e "\nBuilding $pname for: ${concatStringsSep "," targets}"
    gomobile bind \
      ${concatStringsSep " " goBuildFlags} \
      -ldflags="$ldflags" \
      -target=${concatStringsSep "," targets} \
      ${optionalString isAndroid "-androidapi=${platformVersion}" } \
      ${optionalString isIOS "-iosversion=${platformVersion}" } \
     -tags='${optionalString isIOS "nowatchdog"} gowaku_skip_migrations gowaku_no_rln' \
      -o ${outputFileName} \
      ${goPackagePath}/mobile

    runHook postBuild
  '';

  installPhase = ''
    mkdir -p $out
    cp -r ${outputFileName} $out/
    ${if isAndroid then "cp -r ${pkgs.nwaku.libwaku-android-arm64}/libwaku.aar $out;" else ""}
  '';

  # Drop govers from disallowedReferences.
  dontRenameImports = true;
  # Replace hardcoded paths to go package in /nix/store.
  preFixup = optionalString isIOS ''
    find $out -type f -exec \
      remove-references-to -t $disallowedReferences '{}' + || true
  '';
}
