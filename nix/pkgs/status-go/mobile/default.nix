{ 
  self,
  pkgs,
  meta,
  version,
  goBuildLdFlags,
}:
{
  android = {abis ? [ "arm64-v8a" ]}: pkgs.callPackage ./build.nix {
    platform = "android";
    platformVersion = "23";
    # Hide different arch naming in gomobile from Android builds.
    targets = let
      abiMap = {
        "armeabi-v7a" = "android/arm";
        "arm64-v8a"   = "android/arm64";
        "x86"         = "android/386";
        "x86_64"      = "android/amd64";
        };
      in map (arch: abiMap."${arch}") abis;
    outputFileName = "status-go-${self.shortRev or self.dirtyShortRev}.aar";
    inherit self pkgs meta version goBuildLdFlags;
  };

  ios = {targets ? [ "ios/arm64" "iossimulator/amd64"]}: pkgs.callPackage ./build.nix {
    platform = "ios";
    platformVersion = "11.0";
    outputFileName = "Statusgo.xcframework";
    inherit self pkgs meta version goBuildLdFlags targets;
  };
}
