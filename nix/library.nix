{ pkgs ? import <nixpkgs> { }
, src ? ./.. }

pkgs.buildGoModule {
  pname = "status-go-library";
  version = pkgs.lib.fileContents ../VERSION;

  inherit src;
  vendorSha256 = null; # Not necessary, vendor folder exists.
  doCheck = false;

  phases = ["unpackPhase" "configurePhase" "buildPhase"];

  preBuild = ''
    go run cmd/library/*.go > $NIX_BUILD_TOP/main.go
  '';

  # Build the Go library
  buildPhase = ''
    runHook preBuild
    go build -buildmode=c-archive -o $out/libstatus.a $NIX_BUILD_TOP/main.go
    runHook postBuild
  '';
}
