# This file defines custom shells as well as shortcuts
# for accessing more nested shells.
{ config ? {}
, pkgs ? import ./pkgs.nix { inherit config; } }:

let
  inherit (pkgs) lib mkShell callPackage;
  default = callPackage ./shell.nix { };

  shells = {
    inherit default;
  };
in
  shells
