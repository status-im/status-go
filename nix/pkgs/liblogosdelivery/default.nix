{ lib, stdenv, xcbuild, darwin, liblogosdelivery, sealScript }:
# The sealed static liblogosdelivery the Makefile builds, from logos-delivery's
# own package: without RLN, and with only its C and kernel APIs global.
let
  leopardFlags = "-DCMAKE_BUILD_TYPE=Release -DENABLE_OPENMP=off"
    + " -DCOMPILER_SUPPORTS_MARCH_NATIVE=FALSE -DCMAKE_POSITION_INDEPENDENT_CODE=ON"
    + lib.optionalString stdenv.hostPlatform.isx86 " -DCMAKE_CXX_FLAGS=-mssse3";
in
(liblogosdelivery.override {
  enablePostgres = false;
  enableRln = false;
  extraNimArgs = [
    "--define:useMalloc"
    "--passC:-fno-common"
    "--define:\"LeopardCmakeFlags=${leopardFlags}\""
  ] ++ lib.optional stdenv.hostPlatform.isLinux "--passC:-fno-gnu-unique";
}).overrideAttrs (old: {
  pname = "liblogosdelivery-sealed";

  nativeBuildInputs = old.nativeBuildInputs
    ++ lib.optionals stdenv.isDarwin [ darwin.cctools xcbuild ];

  installPhase = ''
    runHook preInstall
    mkdir -p $out/lib $out/include
    bash ${sealScript} "logosdelivery_ waku_" "$NIMCACHE" $out/lib/liblogosdelivery.a
    cp library/liblogosdelivery.h library/liblogosdelivery_kernel.h $out/include/
    runHook postInstall
  '';
})
