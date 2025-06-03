{ self, pkgs }:

let
  inherit (pkgs.lib) attrValues mapAttrs;

  # Metadata common to all builds of status-go
  meta = {
    description = "The Status Go module that consumes go-ethereum.";
    license = pkgs.lib.licenses.mpl20;
    platforms = with pkgs.lib.platforms; linux ++ darwin;
  };

  version = self.rev or self.dirtyRev;

  # Params to be set at build time, important for About section and metrics
  goBuildParams = {
    GitCommit = self.rev or self.dirtyRev;
    Version = version;
    # FIXME: This should be moved to status-go config.
    IpfsGatewayURL = "https://ipfs.status.im/";
  };

  # These are necessary for status-go to show correct version
  paramsLdFlags = attrValues (mapAttrs (name: value:
    "-X github.com/status-im/status-go/params.${name}=${value}"
  ) goBuildParams);

  goBuildLdFlags = paramsLdFlags ++ [
    "-s" # -s disabled symbol table
    "-w" # -w disables DWARF debugging information
  ];
in rec {
  mobile = pkgs.callPackage ./mobile {
    inherit self pkgs meta goBuildLdFlags version;
  };

  library = pkgs.callPackage ./library {
    inherit self pkgs meta version;
  };
}
