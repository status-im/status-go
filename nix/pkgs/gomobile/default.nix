{
  stdenv,
  lib,
  fetchgit,
  buildGo122Module,
  zlib,
  makeWrapper,
  xcodeenv,
  androidenv,
  xcodeWrapperArgs ? { },
  xcodeWrapper ? xcodeenv.composeXcodeWrapper xcodeWrapperArgs,
  withAndroidPkgs ? true,
  androidPkgs ? (
    androidenv.composeAndroidPackages {
      includeNDK = true;
    }
  ),
}:

buildGo122Module {
  pname = "gomobile";
  version = "0-unstable-2024-12-13";

  src = fetchgit {
    name = "gomobile";
    url = "https://go.googlesource.com/mobile";
    rev = "a87c1cf6cf463f0d4476cfe0fcf67c2953d76e7c";
    hash = "sha256-7j4rdmCZMC8tn4vAsC9x/mMNkom/+Tl7uAY+5gkSvfY=";
  };

  vendorHash = "sha256-6ycxEDEE0/i6Lxo0gb8wq3U2U7Q49AJj+PdzSl57wwI=";

  CGO_ENABLED = "1";

  preBuild = lib.optionalString stdenv.hostPlatform.isDarwin ''
    # to fix : error: unable to find sdk: 'iphoneos'
    export DEVELOPER_DIR="/Applications/Xcode.app/Contents/Developer";
    export SDKROOT="$(/usr/bin/xcrun --sdk macosx --show-sdk-path)";
    if ! xcrun --sdk iphoneos --show-sdk-path >/dev/null 2>&1; then
      echo "iOS SDK not found. Ensure Xcode is installed: sudo xcode-select --switch /Applications/Xcode.app"
    fi
  '';

  subPackages = [
    "bind"
    "cmd/gobind"
    "cmd/gomobile"
  ];

  # Fails with: go: cannot find GOROOT directory
  doCheck = false;

  nativeBuildInputs = [ makeWrapper ] ++ lib.optionals stdenv.hostPlatform.isDarwin [ xcodeWrapper ];

  # Prevent a non-deterministic temporary directory from polluting the resulting object files
  postPatch = ''
    substituteInPlace cmd/gomobile/env.go --replace-quiet \
      'tmpdir, err = ioutil.TempDir("", "gomobile-work-")' \
      'tmpdir = filepath.Join(os.Getenv("NIX_BUILD_TOP"), "gomobile-work")'
    substituteInPlace cmd/gomobile/init.go --replace-quiet \
      'tmpdir, err = ioutil.TempDir(gomobilepath, "work-")' \
      'tmpdir = filepath.Join(os.Getenv("NIX_BUILD_TOP"), "work")'

    # To fix unable to import bind: no Go package in golang.org/x/mobile/bind
    substituteInPlace cmd/gomobile/init.go --replace \
      'golang.org/x/mobile/cmd/gobind@latest' \
      'golang.org/x/mobile/cmd/gobind'
  '';

  # Necessary for GOPATH when using gomobile.
  postInstall = ''
    mkdir -p $out/src/golang.org/x
    ln -s $src $out/src/golang.org/x/mobile
  '';

  # to fix : error: unable to find sdk: 'iphoneos'
  postFixup = ''
    for prog in gomobile gobind; do
      wrapProgram $out/bin/$prog \
        --suffix GOPATH : $out \
        --prefix LD_LIBRARY_PATH : "${lib.makeLibraryPath [ zlib ]}" \
        ${lib.optionalString stdenv.hostPlatform.isDarwin ''
          --set DEVELOPER_DIR "/Applications/Xcode.app/Contents/Developer" \
          --prefix PATH : "/usr/bin" \
          --run 'export SDKROOT="$(/usr/bin/xcrun --sdk macosx --show-sdk-path)"' \
        ''} \
        ${lib.optionalString withAndroidPkgs ''
          --prefix PATH : "${androidPkgs.androidsdk}/bin" \
          --set-default ANDROID_HOME "${androidPkgs.androidsdk}/libexec/android-sdk"
        ''}
    done
  '';

  meta = {
    description = "Tool for building and running mobile apps written in Go";
    homepage = "https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile";
    license = with lib.licenses; [ bsd3 ];
    maintainers = with lib.maintainers; [ jakubgs ];
  };
}