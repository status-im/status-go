{
  pkgs,
  self,
  meta,
  version,
  stdenv,
}:

let
  optionalString = pkgs.lib.optionalString;

in
pkgs.buildGoModule {
  pname = "status-go";
  src = builtins.path { path = ./../../../..; name = "status-go-library"; };
  vendorHash = null;

  inherit meta version;

  nativeBuildInputs =
    let
      # Fixes fatal: not a git repository (or any of the parent directories): .git
      fakeGit = pkgs.writeScriptBin "git" "echo ${version}";
    in
    with pkgs; [
      mockgen
      protoc-gen-go
      protobuf3_24
      fakeGit
    ];

  phases = ["unpackPhase" "configurePhase" "patchPhase" "buildPhase"];

  # https://pkg.go.dev/net#hdr-Name_Resolution
  # https://github.com/status-im/status-mobile/issues/19736
  # https://github.com/status-im/status-mobile/issues/19581
  # TODO: try removing when go is upgraded to 1.22
  GODEBUG = "netdns=cgo+2";

  # Since go 1.21 status-go compiled library includes references to cgo runtime.
  # FIXME: Remove this when go 1.23 or later versions fix this madness.
  allowGoReference = true;

  preBuild = ''
    go run cmd/library/*.go > $NIX_BUILD_TOP/main.go
    make generate SHELL=$SHELL GO111MODULE=on GO_GENERATE_CMD='go generate'

    export CGO_CFLAGS="-I${pkgs.lib-sds-pkg}/include"
    export CGO_LDFLAGS="-L${pkgs.lib-sds-pkg}/lib"
  '';

  # Build the Go library
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  buildPhase = ''
    # make sure Go modules build correctly
    export GOPATH=$NIX_BUILD_TOP/go
    export GO111MODULE=on

    # Patch env.sh now that nim-sds exists
    echo "Patching env.sh to use Nix Nim..."
    env_sh="vendor/github.com/waku-org/sds-go-bindings/third_party/nim-sds/vendor/nimbus-build-system/scripts/env.sh"
    if [ -f "$env_sh" ]; then
      substituteInPlace "$env_sh" \
        --replace-warn "/vendor/Nim/bin/nim" "${pkgs.nim}/bin/nim"
    fi

    # Avoid building nim-sds because the lib and header comes from nim-sds flake in patchPhase
    substituteInPlace Makefile \
      --replace-warn "\$(MAKE) -C \$(NIM_SDS_SOURCE_DIR) libsds USE_SYSTEM_NIM=\$(USE_SYSTEM_NIM) SHELL=/bin/bash" "echo 'Skipping nim-sds build...'"

    runHook preBuild
    go build \
      -buildmode='c-archive' \
      ${optionalString stdenv.isDarwin "-ldflags=-extldflags=-lresolv"} \
      -tags='gowaku_skip_migrations gowaku_no_rln ${optionalString stdenv.isDarwin "netgo"}' \
      -o "$out/libstatus.a" \
      $NIX_BUILD_TOP/main.go
    runHook postBuild
  '';
}
