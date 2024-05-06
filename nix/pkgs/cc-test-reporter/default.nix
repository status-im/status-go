{ lib, stdenv, fetchurl }:

let
  inherit (stdenv) isLinux isDarwin isWindows;

  platform =
    if isLinux then "linux" else
    if isDarwin then "darwin" else
    if isWindows then "windows" else
    throw "Unsupported platform: ${stdenv.hostPlatform.system}";

in stdenv.mkDerivation rec {
  pname = "cc-test-reporter";
  version = "0.11.1";

  src = fetchurl {
    url = "https://codeclimate.com/downloads/test-reporter/test-reporter-${version}-${platform}-amd64";
    hash = lib.getAttr platform {
      darwin = "sha256-uO9aRL3cJe+KCoC+uN6cBQy8xGQHim6h5Qzw36QO7EY=";
      linux = "sha256-ne79mW3w9tHJ+3lAWzluuRp6yjWsy4lpdV/KpmjaTa0=";
      windows = "sha256-8pn8csW9l5xMerZWAwIwWcrO7OLNWEM03yPEMMllaak=";
    };
   };

  dontUnpack = true;

  installPhase = ''
    install -D $src $out/bin/cc-test-reporter
    chmod +x $out/bin/cc-test-reporter
  '';
}
