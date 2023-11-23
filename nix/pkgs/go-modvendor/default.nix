{ buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "go-modvendor";
  version =  "0.5.0";
  vendorHash = null;

  src = fetchFromGitHub rec {
    owner = "goware";
    repo = "modvendor";
    rev = "v${version}";
    hash = "sha256-6Zht3XukH6rZaiz9aNQI+SXuonqw7k2LiElLPH2Zkwo=";
  };
}
