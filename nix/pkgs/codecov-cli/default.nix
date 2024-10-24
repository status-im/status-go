{ lib, stdenv, fetchurl }:

let
  platform = lib.getAttr builtins.currentSystem {
    aarch64-linux = "linux-arm64";
    x86_64-linux = "linux";
    aarch64-darwin = "macos"; # There's no arm64 build for macOS, amd64 works on both
    x86_64-darwin = "macos";
  };

in stdenv.mkDerivation rec {
  pname = "codecov";
  version = "0.7.4";

  src = fetchurl {
    url = "https://cli.codecov.io/v${version}/${platform}/codecov";
    hash = lib.getAttr builtins.currentSystem {
      aarch64-darwin = "sha256-CB1D8/zYF23Jes9sd6rJiadDg7nwwee9xWSYqSByAlU=";
      x86_64-darwin = "sha256-CB1D8/zYF23Jes9sd6rJiadDg7nwwee9xWSYqSByAlU=";
      x86_64-linux = "sha256-65AgCcuAD977zikcE1eVP4Dik4L0PHqYzOO1fStNjOw=";
      aarch64-linux = "sha256-hALtVSXY40uTIaAtwWr7EXh7zclhK63r7a341Tn+q/g=";
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
