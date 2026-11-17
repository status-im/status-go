# Nix with the dependencies of status-go-library already built, see scripts/nix-image.sh.
FROM nixos/nix:2.33.1@sha256:d5cce2440bda1f966357732c06d86cb92368069fb52dfb6b2bae8725eea488a5

LABEL org.opencontainers.image.source=https://github.com/status-im/status-go
ENV NIX_CONFIG="experimental-features = nix-command flakes"
# The container runs as root on someone else's checkout, and Nix reads git settings from a file only.
RUN printf '[safe]\n\tdirectory = *\n' > /root/.gitconfig

COPY . /flake
RUN bash /flake/nix/scripts/ci-image.sh /flake
