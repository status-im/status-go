{
  pkgs,
  self,
  meta,
  version,
  stdenv,
}:

let
  optionalString = pkgs.lib.optionalString;
  codexVersion = "v0.0.25";
  arch =
    if stdenv.hostPlatform.isx86_64 then "amd64"
    else if stdenv.hostPlatform.isAarch64 then "arm64"
    else stdenv.hostPlatform.arch;
  os = if stdenv.isDarwin then "macos" else "Linux";
  hash = 
    if stdenv.hostPlatform.isDarwin 
    # nix store prefetch-file --json --unpack https://github.com/codex-storage/codex-go-bindings/releases/download/v0.0.25/codex-macos-arm64.zip | jq -r .hash
    then "sha256-0AwwTom5i8v+hG81ikKjXWVeq7/v/FNVyb+3clH/V1Y="
    # nix store prefetch-file --json --unpack https://github.com/codex-storage/codex-go-bindings/releases/download/v0.0.25/codex-Linux-amd64.zip | jq -r .hash
    else "sha256-P1w1XvWsg/ZPg8VZfd52hffI2u4SIIWekIWVP79YnCc=";

  # Pre-fetch libcodex to avoid network during build
  codexLib = pkgs.fetchzip {
    url = "https://github.com/codex-storage/codex-go-bindings/releases/download/${codexVersion}/codex-${os}-${arch}.zip";
    hash = hash;
    stripRoot = false;
  };

in pkgs.buildGoModule {
  pname = "status-go";
  src = builtins.path { path = ./../../../..; name = "status-go-library"; };
  vendorHash = "sha256-bImoWkSlJw2oRgtXh4e76gxAmgzaFqGvhTzsH8dYwWY=";

  inherit meta version;

  nativeBuildInputs = let
    # Fixes fatal: not a git repository (or any of the parent directories): .git
    fakeGit = pkgs.writeScriptBin "git" "echo ${version}";
  in
    with pkgs; [
      mockgen
      protoc-gen-go
      protobuf3_24
      fakeGit
  ];

  phases = ["unpackPhase" "configurePhase" "buildPhase"];

  # https://pkg.go.dev/net#hdr-Name_Resolution
  # https://github.com/status-im/status-mobile/issues/19736
  # https://github.com/status-im/status-mobile/issues/19581
  # TODO: try removing when go is upgraded to 1.22
  GODEBUG = "netdns=cgo+2";

  # Since go 1.21 status-go compiled library includes references to cgo runtime.
  # FIXME: Remove this when go 1.23 or later versions fix this madness.
  allowGoReference = true;

  preBuild = ''
    export LIBS_DIR="${codexLib}"
    export NIM_SDS_INC_DIR="${pkgs.lib-sds-pkg}/include"
    export NIM_SDS_LIB_DIR="${pkgs.lib-sds-pkg}/lib"
    export GO111MODULE=on
    export GO_GENERATE_CMD='go generate'
    NO_NETWORK=1 LIBS_DIR="$LIBS_DIR" make generate
  '';

  # Build the Go library
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  buildPhase = ''
    runHook preBuild
    CGO_ENABLED=1 \
    CGO_CFLAGS="-I$LIBS_DIR" \
    CGO_LDFLAGS="-L$LIBS_DIR -lcodex -Wl,-rpath,$LIBS_DIR" \
    make statusgo-library \
        STATUS_GO_BINDINGS_PATH="$NIX_BUILD_TOP" \
        STATUS_GO_LIBRARY_OUT="$out"
    runHook postBuild
  '';
}
