FROM debian:bookworm-slim

RUN apt-get update && apt-get install -yq --no-install-recommends --fix-missing \
    ca-certificates \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r jenkins --gid 1001 \
    && useradd -r -m -g jenkins --uid 1001 -d /home/jenkins jenkins

RUN groupadd -g 999 docker && \
    usermod -a -G docker jenkins

USER jenkins

ENV PATH="${PATH}:/nix/var/nix/profiles/default/bin"
ENV NIX_REMOTE=daemon
ENV HOME=/home/jenkins

ENTRYPOINT [""]
