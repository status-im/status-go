# Nix with the dependencies of the status-go-library package already built.
# CI runs `nix build` in it, see scripts/nix-image.sh. The build context is
# flake.nix, flake.lock and nix/, nothing else.
FROM nixos/nix:2.33.1@sha256:d5cce2440bda1f966357732c06d86cb92368069fb52dfb6b2bae8725eea488a5

ENV NIX_CONFIG="experimental-features = nix-command flakes"

COPY . /flake
RUN bash /flake/nix/scripts/ci-image.sh /flake
