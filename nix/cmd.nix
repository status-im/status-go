{ pkgs ? import <nixpkgs> { }
, src ? ./..
, command ? "statusd"
, os ? "linux"
, arch ? "amd64" }:

pkgs.buildGoModule {
  pname = "status-go-${command}-${arch}-${os}";
  version = pkgs.lib.fileContents ../VERSION;

  inherit src;
  vendorSha256 = null; # Not necessary, vendor folder exists.
  doCheck = false;

  subPackages = ["cmd/${command}"];
  GOOS = os;
  GOARCH = arch;
}
