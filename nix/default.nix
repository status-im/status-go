{
    config ? {},
    pkgs ? import ./pkgs.nix { inherit config; }
}:
 let
  # put all main targets and shells together for easy import
  shells = pkgs.callPackage ./shells.nix { };
in {
  inherit pkgs shells;
}
