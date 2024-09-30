{ buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "go-generate-fast";
  version = "0.3.0";

  subPackages = [ "." ];

  src = fetchFromGitHub rec {
    owner = "oNaiPs";
    repo = "go-generate-fast";
    rev = "v${version}";
    hash = "sha256-NMGXOI3y3PGt+hrHhOsugACL8c5LIzpwwdt+Ne0MkY8=";
  };
  vendorHash = "sha256-8nmnTuDZvnFEPQAxOv19gUgHy6FpI3HLRtqLLob+zrE=";
}