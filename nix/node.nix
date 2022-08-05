{ buildGoModule, lib, src }:

buildGoModule {
  pname = "status-go";
  version = lib.fileContents ../VERSION;

  inherit src;
  vendorSha256 = null; # Not necessary when vendor folder exists.
  doCheck = false;

  subPackages = ["cmd/statusd"];

  #installPhase = ''
  #  set -x
  #  ls -l build/bin
  #  mkdir -p $out/bin
  #  mv build/bin/statusd $out/bin/statusd
  #'';
}
