{
  description = "Nix flake for status-go.";

  inputs.nixpkgs.url = github:NixOS/nixpkgs/nixos-21.11;
  inputs.flake-utils.url = github:numtide/flake-utils;

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system: 
      let
        #pkgs = import nixpkgs { system = "x86_64-linux"; };
        pkgs = nixpkgs.legacyPackages.${system};
        inherit (pkgs.lib) cartesianProductOfSets listToAttrs forEach;

        # All viable builds of status-go command line tools.
        builds = cartesianProductOfSets {
          command = ["statusd" "bootnode" "node-canary" "ping-community"];
          os = ["linux" "darwin" "android" "ios"];
          arch = ["386" "amd64" "arm64"];
        };

        # Helper for building commands
        buildCommand = config: pkgs.callPackage ./nix/cmd.nix ({ src = self; } // config);
      in rec {
        packages = flake-utils.lib.flattenTree (listToAttrs (forEach builds (
          config: {
            name = "${config.command}-${config.arch}-${config.os}";
            value = buildCommand config;
          }
        )));
        #defaultPackage = packages."statusd-amd64-linux";
        defaultPackage = packages."statusd-arm64-android";
      }
    );
}
