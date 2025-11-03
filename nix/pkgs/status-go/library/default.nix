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
    then "sha256-vlQu7mCGuDL+dKBsD1yZ+PZenZYtmM2TxjU5b/Gi1pQ="
    # nix store prefetch-file --json --unpack https://github.com/codex-storage/codex-go-bindings/releases/download/v0.0.25/codex-Linux-amd64.zip | jq -r .hash
    else "sha256-SVJsnEZF5Bkh3zBWBCD1klpAb/Q3bePX8HB7NCeSY20=";

  # Pre-fetch libcodex to avoid network during build
  codexLib = pkgs.fetchzip {
    url = "https://github.com/codex-storage/codex-go-bindings/releases/download/${codexVersion}/codex-${os}-${arch}.zip";
    hash = hash;
    stripRoot = false;
  };

in pkgs.buildGoModule {
  pname = "status-go";
  src = builtins.path { path = ./../../../..; name = "status-go-library"; };
  vendorHash = null;

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
    go run cmd/library/*.go > $NIX_BUILD_TOP/main.go
    NO_NETWORK=1 LIBS_DIR="$LIBS_DIR" make generate SHELL=$SHELL GO111MODULE=on GO_GENERATE_CMD='go generate'
  '';

  # Build the Go library
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  buildPhase = ''
    runHook preBuild
    CGO_ENABLED=1 \
    CGO_CFLAGS="-I$LIBS_DIR" \
    CGO_LDFLAGS="-L$LIBS_DIR -lcodex -Wl,-rpath,$LIBS_DIR" \
    go build \
      -buildmode='c-archive' \
      ${optionalString stdenv.isDarwin "-ldflags=-extldflags=-lresolv"} \
      -tags='gowaku_skip_migrations gowaku_no_rln ${optionalString stdenv.isDarwin "netgo"}' \
      -o "$out/libstatus.a" \
      $NIX_BUILD_TOP/main.go
    runHook postBuild
  '';
}
