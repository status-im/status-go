{ lib, stdenv, fetchurl, system }:

let
  platform = lib.getAttr system {
    aarch64-linux = "linux-arm64";
    x86_64-linux = "linux";
    aarch64-darwin = "macos"; # There's no arm64 build for macOS, amd64 works on both
    x86_64-darwin = "macos";
  };

in stdenv.mkDerivation rec {
  pname = "codecov";
  version = "11.1.0";

  src = fetchurl {
    url = "https://cli.codecov.io/v${version}/${platform}/codecov";
    hash = lib.getAttr system {
      aarch64-darwin = "sha256-Arm/CUJ/QdGwVhA5WWt0xgB2hsGogn+epeAVKQZ+maQ=";
      x86_64-darwin = "sha256-Arm/CUJ/QdGwVhA5WWt0xgB2hsGogn+epeAVKQZ+maQ=";
      x86_64-linux = "sha256-XMiVGIZOquSfOBnkK6NT+aPGdpEtRHI1HTGXBkPxpJo=";
      aarch64-linux = "sha256-e4hqsXUMxL+AyADIv1CunhDHeohuSKTRXtxaCWXj2qg=";
    };
   };

  dontUnpack = true;
  stripDebug = false;
  dontStrip = true; # This is to prevent `Could not load PyInstaller's embedded PKG archive from the executable` error

  installPhase = ''
    runHook preInstall
    install -D $src $out/bin/codecov
    chmod +x $out/bin/codecov
    runHook postInstall
  '';

  meta = with lib; {
    description = "Codecov CLI tool to upload coverage reports";
    homepage = "https://docs.codecov.com/docs/the-codecov-cli";
    license = licenses.asl20;
    mainProgram = "codecov";
    platforms = ["aarch64-linux" "x86_64-linux" "aarch64-darwin" "x86_64-darwin"];
  };
}
