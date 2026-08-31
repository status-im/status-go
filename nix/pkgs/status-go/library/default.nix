{
  pkgs,
  self,
  meta,
  version,
  stdenv,
}:

let
  optionalString = pkgs.lib.optionalString;

  # Fixes fatal: not a git repository (or any of the parent directories): .git
  fakeGit = pkgs.writeScriptBin "git" "echo ${version}";
in pkgs.buildGoModule {
  pname = "status-go";
  src = builtins.path { path = ./../../../..; name = "status-go-library"; };

  # WARNING: Needs to be updated when go.mod is changed.
  vendorHash = "sha256-KTdd//keDwVjw1Jb9+Vt4zLoNx8Us4aHxP449HuROyM=";

  inherit meta version;

  nativeBuildInputs = with pkgs; [
    mockgen
    protobuf_36
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

  # Fix /homeless-shelter error and provide commit without affecting go modules.
  preConfigure = ''
    export HOME=$TMPDIR
    export PATH="${fakeGit}/bin:$PATH"
  '';

  # Code generation should be run before buildPhase because buildGoModule
  # performs deps check before buildPhase, and will fail without generated files.
  preBuild = ''
    make generate GO_GENERATE_CMD='go generate'
  '';

  # Build the Go library
  #
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  #
  # Also set CLEANUP_GENERATED_FILES_DRY_RUN=true to avoid running cleanup_generated_files.sh script,
  # which is not available at this phase, because buildGoModule only copies Go files.
  buildPhase = ''
    # this line removes a bug where value of $HOME is set to a non-writable /homeless-shelter dir
    export HOME=$TMPDIR
    make statusgo-library \
        USE_LOGOS_STORAGE=true \
        NIM_SDS_INC_DIR="${pkgs.libsds}/include" \
        NIM_SDS_LIB_DIR="${pkgs.libsds}/lib" \
        LOGOS_STORAGE_LIB_DIR="${pkgs.libstorage}/lib" \
        LOGOS_STORAGE_INC_DIR="${pkgs.libstorage}/include" \
        STATUS_GO_BINDINGS_PATH="$NIX_BUILD_TOP" \
        STATUS_GO_LIBRARY_OUT="$out" \
        CLEANUP_GENERATED_FILES=false \
        GO_GENERATE_CMD='go generate'
    runHook postBuild
  '';
}
