{ buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "go-gencodec";

  src = fetchFromGitHub rec {
    owner = "fjl";
    repo = "gencodec";
    rev = "f9840df";
    hash = "sha256-6Zht3XukH6rZaiz9aNQI+SXuonqw7k2LiElLPH2Zkwo=";
  };
}
