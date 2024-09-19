# This file controls the pinned version of nixpkgs we use for our Nix environment
# as well as which versions of package we use, including their overrides.
{ config ? { } }:

let
  # For testing local version of nixpkgs
  #nixpkgsSrc = (import <nixpkgs> { }).lib.cleanSource "/home/jakubgs/work/nixpkgs";

  # We follow the master branch of official nixpkgs.
  nixpkgsSrc = builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs/archive/48d567fc7b299de90f70a48e4263e31f690ba03e.tar.gz";
    sha256 = "sha256:0zynbk52khdfhg4qfv26h3r5156xff5p0cga2cin7b07i7lqminh";
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
