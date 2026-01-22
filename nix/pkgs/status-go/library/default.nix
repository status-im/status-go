{
  pkgs,
  self,
  meta,
  version,
  stdenv,
}:

let
  optionalString = pkgs.lib.optionalString;
  logosStorageVersion = "v0.0.30";
  arch =
    if stdenv.hostPlatform.isx86_64 then "amd64"
    else if stdenv.hostPlatform.isAarch64 then "arm64"
    else stdenv.hostPlatform.arch;
  os = if stdenv.isDarwin then "macos" else "Linux";
  hash = 
    if stdenv.hostPlatform.isDarwin 
    # nix store prefetch-file --json --unpack https://github.com/logos-storage/logos-storage-go-bindings/releases/download/${logosStorageVersion}/storage-macos-arm64.zip | jq -r .hash
    then "sha256-GcerkH8izZ5QHG5ARNNrM1fktaeBKjF6AGNsA6vxVj0="
    # nix store prefetch-file --json --unpack https://github.com/logos-storage/logos-storage-go-bindings/releases/download/${logosStorageVersion}/storage-linux-amd64.zip | jq -r .hash
    else "sha256-sYhbgBN0LNA7YhmBigPwo1h34QTADTxFGjO8QAw8m18=";

  # Pre-fetch libstorage to avoid network during build
  logosStorageLib = pkgs.fetchzip {
    url = "https://github.com/logos-storage/logos-storage-go-bindings/releases/download/${logosStorageVersion}/storage-${os}-${arch}.zip";
    hash = hash;
    stripRoot = false;
  };

in pkgs.buildGoModule {
  pname = "status-go";
  src = builtins.path { path = ./../../../..; name = "status-go-library"; };
  vendorHash = "sha256-WCYruo0mEMzm/G8LGZCZRftfbOGxEyXp5BiUPOfpJCY=";

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

  # Code generation should be run before buildPhase because buildGoModule
  # performs dependency inspection before buildPhase, and will fail if generated files are missing.
  preBuild = ''
    # this line removes a bug where value of $HOME is set to a non-writable /homeless-shelter dir
    export HOME=$TMPDIR
    export LIBS_DIR="${logosStorageLib}"

    make generate \
        NO_NETWORK=1 \
        NIM_SDS_INC_DIR="${pkgs.lib-sds-pkg}/include" \
        NIM_SDS_LIB_DIR="${pkgs.lib-sds-pkg}/lib" \
        GO111MODULE=on \
        GO_GENERATE_CMD='go generate'
  '';

  # Build the Go library
  #
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  #
  # Also set CLEANUP_GENERATED_FILES_DRY_RUN=true to avoid running cleanup_generated_files.sh script,
  # which is not available at this phase, because buildGoModule only copies Go files.
  buildPhase = ''
    runHook preBuild
    # this line removes a bug where value of $HOME is set to a non-writable /homeless-shelter dir
    export HOME=$TMPDIR
    CGO_ENABLED=1 \
    CGO_CFLAGS="-I$LIBS_DIR -I$NIM_SDS_INC_DIR" \
    CGO_LDFLAGS="-L$LIBS_DIR -lstorage -Wl,-rpath,$LIBS_DIR -L$NIM_SDS_LIB_DIR -lsds" \
    make statusgo-library \
        NO_NETWORK=1 \
        NIM_SDS_INC_DIR="${pkgs.lib-sds-pkg}/include" \
        NIM_SDS_LIB_DIR="${pkgs.lib-sds-pkg}/lib" \
        STATUS_GO_BINDINGS_PATH="$NIX_BUILD_TOP" \
        STATUS_GO_LIBRARY_OUT="$out" \
        CLEANUP_GENERATED_FILES=false \
        GO_GENERATE_CMD='go generate'
    runHook postBuild
  '';
}
