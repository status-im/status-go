{ stdenv, nim, nim-unwrapped-2_2, fetchurl, src, lib, darwin }:
let
  # Match the public library's CI compiler without changing any SDS toolchain.
  compiler = nim.override {
    nim-unwrapped-2_2 = nim-unwrapped-2_2.overrideAttrs (old: {
      version = "2.2.10";
      src = fetchurl {
        url = "https://nim-lang.org/download/nim-2.2.10.tar.xz";
        hash = "sha256-eVe37QBCBrzxC8xPO0dEFTh45i8kMVUqmo6dP0Do1dU=";
      };
      # This optional store-path shortening patch targets older compiler code.
      patches = builtins.filter (p: builtins.baseNameOf p != "extra-mangling-2.patch") old.patches;
      kochArgs = builtins.filter (arg: arg != "-d:nativeStacktrace") old.kochArgs;
    });
  };
in
stdenv.mkDerivation {
  pname = "libtkl";
  version = "9a9f2fb53515";
  inherit src;
  nativeBuildInputs = [ compiler ] ++ lib.optionals stdenv.isDarwin [ darwin.cctools ];
  postPatch = ''
    patchShebangs --build scripts
  '';
  buildPhase = ''
    export HOME="$TMPDIR"
    export NIM=nim
    export EXTRA_NIMFLAGS="--cc:${if stdenv.isDarwin then "clang" else "gcc"}"
    make isolate
  '';
  installPhase = ''
    mkdir -p "$out/lib" "$out/include" "$out/share/licenses/libtkl"
    cp build/libtkl_isolated.a "$out/lib/libtkl.a"
    cp abi/tkl.h "$out/include/"
    for dep in vendor/*; do
      mkdir -p "$out/share/licenses/libtkl/$(basename "$dep")"
      cp "$dep"/LICENSE-* "$out/share/licenses/libtkl/$(basename "$dep")/"
    done
  '';
  meta.platforms = lib.platforms.linux ++ lib.platforms.darwin;
}
