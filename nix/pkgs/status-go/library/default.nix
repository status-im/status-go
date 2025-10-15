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
  vendorHash = null;

  inherit meta version;

  nativeBuildInputs = let
    # Fixes fatal: not a git repository (or any of the parent directories): .git
    fakeGit = pkgs.writeScriptBin "git" "echo ${version}";
  in
    with pkgs; [
      which
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

  # TODO: This is wrong, we shouldn't be building `nim-sds` as part of this derivtion.
  # Instead we need to fix flake.nix in nim-sds and pull already built lib from that.
  NIX_DEBUG = 1;
  patchPhase = ''
    pushd vendor/github.com/waku-org/sds-go-bindings
    mkdir -p third_party
    cp -r ${pkgs.nim-sds-src} third_party/nim-sds
    chmod 775 -R third_party/nim-sds
    popd
  '';

  preBuild = ''
    go run cmd/library/*.go > $NIX_BUILD_TOP/main.go
    make generate SHELL=$SHELL GO111MODULE=on GO_GENERATE_CMD='go generate'
  '';

  # Build the Go library
  # ld flags and netgo tag are necessary for integration tests to work on MacOS
  # https://github.com/status-im/status-mobile/issues/20135
  buildPhase = ''
    # Set Go cache inside writable directory
    export GOCACHE=$NIX_BUILD_TOP/.gocache
    mkdir -p $GOCACHE

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
