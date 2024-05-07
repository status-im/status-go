{ lib, stdenv, fetchurl }:

let
  platform = lib.getAttr builtins.currentSystem {
    aarch64-linux = "linux-arm64";
    x86_64-linux = "linux-amd64";
    aarch64-darwin = "darwin-amd64"; # There's no arm64 build for macOS, amd64 works on both
    x86_64-darwin = "darwin-amd64";
  };

in stdenv.mkDerivation rec {
  pname = "cc-test-reporter";
  version = "0.11.1";

  src = fetchurl {
    url = "https://codeclimate.com/downloads/test-reporter/test-reporter-${version}-${platform}";
    hash = lib.getAttr builtins.currentSystem {
      aarch64-linux = "sha256-b6rTiiKZiVxoR/aQaxlqG6Ftt7sqyAKXgO9EG6/sKck=";
      x86_64-linux = "sha256-ne79mW3w9tHJ+3lAWzluuRp6yjWsy4lpdV/KpmjaTa0=";
      aarch64-darwin = "sha256-uO9aRL3cJe+KCoC+uN6cBQy8xGQHim6h5Qzw36QO7EY=";
      x86_64-darwin = "sha256-uO9aRL3cJe+KCoC+uN6cBQy8xGQHim6h5Qzw36QO7EY=";
    };
   };

  dontUnpack = true;

  installPhase = ''
    runHook preInstall
    install -D $src $out/bin/cc-test-reporter
    chmod +x $out/bin/cc-test-reporter
    runHook postInstall
  '';

  meta = with lib; {
    description = "Code Climate test reporter for sending coverage data";
    homepage = "https://docs.codeclimate.com/docs/configuring-test-coverage";
    license = licenses.mit;
    mainProgram = "cc-test-reporter";
    platforms = ["aarch64-linux" "x86_64-linux" "aarch64-darwin" "x86_64-darwin"];
  };
}
