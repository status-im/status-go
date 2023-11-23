# This file controls the pinned version of nixpkgs we use for our Nix environment
# as well as which versions of package we use, including their overrides.
{ config ? { } }:

let
  # For testing local version of nixpkgs
  #nixpkgsSrc = (import <nixpkgs> { }).lib.cleanSource "/home/jakubgs/work/nixpkgs";

  # We follow the master branch of official nixpkgs.
  nixpkgsSrc = builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/224fd9a362487ab2894dac0df161c84ab1d8880b.tar.gz";
    sha256 = "sha256:1syvl39pi1h8lf5gkd9h7ksn5hp34cj7pa3abr59217kv0bdklhy";
  };

  # Status specific configuration defaults
  defaultConfig = {
      allowUnfree = true;
    android_sdk.accept_license = true;
  };
  # Override some packages and utilities
  pkgsOverlay = import ./overlay.nix;

in
  # import nixpkgs with a config override
  (import nixpkgsSrc) {
    config = defaultConfig // config;
    overlays =  [pkgsOverlay];
  }
