{
  pkgs,
  self,
  meta,
  version,
  stdenv,
}:

let
  optionalString = pkgs.lib.optionalString;
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
    export NIM_SDS_INC_DIR="${pkgs.lib-sds-pkg}/include"
    export NIM_SDS_LIB_DIR="${pkgs.lib-sds-pkg}/lib"
    export GO_GENERATE_CMD='go generate'
    make generate
  '';

  # Build the Go library
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  buildPhase = ''
    runHook preBuild
    make statusgo-library \
        STATUS_GO_BINDINGS_PATH="$NIX_BUILD_TOP" \
        STATUS_GO_LIBRARY_OUT="$out"
    runHook postBuild
  '';
}
