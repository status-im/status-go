{ buildGoModule, fetchFromGitHub }:

buildGoModule rec {
  pname = "go-generate-fast";
  version = "0.3.0";

  src = fetchFromGitHub rec {
    owner = "oNaiPs";
    repo = "go-generate-fast";
    rev = "v${version}";
    hash = "sha256-UXN5gN1SNv3uvBCliJQ+5PSGHRL7RyU6pmZtGUTFsrQ=";
  };
  vendorHash = "sha256-jUlK/A0HxBrIby2C0zYFtnxQX1bgKVyypI3QdH4u/rg=";
}