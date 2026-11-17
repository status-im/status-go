# Nix with the dependencies of status-go-library already built, see scripts/nix-image.sh.
# Nix 2.19 uses Git's submodule checkout, which can fetch pinned commits absent from a configured branch.
FROM nixos/nix:2.19.3@sha256:498fa2d7f2b5cb3891a4edf20f3a8f8496e70865099ba72540494cd3e2942634

LABEL org.opencontainers.image.source=https://github.com/status-im/status-go
ENV NIX_CONFIG="experimental-features = nix-command flakes"
# The container runs as root on someone else's checkout, and Nix reads git settings from a file only.
RUN printf '[safe]\n\tdirectory = *\n' > /root/.gitconfig

COPY . /flake
RUN bash /flake/nix/scripts/ci-image.sh /flake
