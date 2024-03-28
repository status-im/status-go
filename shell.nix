# for passing build optionsm see nix/README
# TODO complet nix/README
{ config ? { } }:

let
  main = import ./nix { inherit config; };
in
  # use the default shell when calling nix-shell without arguments
  main.shells.default
