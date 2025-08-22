FROM statusteam/nim-status-client-build:2.0.4-qt6.9.0

USER root

RUN apt-get update && apt-get install -yq --no-install-recommends --fix-missing \
    sudo \
    curl wget gnupg ca-certificates lsb-release python3-pip python3-venv \
    && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
    | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update && apt-get install -y \
    docker-ce-cli docker-compose-plugin \
    cmake protoc-gen-go \
    mesa-common-dev libglu1-mesa-dev libpcsclite-dev \
    xvfb fluxbox libxft-dev xclip xsel nautilus \
    tesseract-ocr libzbar-dev libopenjp2-7 \
    ruby-dev ruby-bundler leiningen ghp-import git-lfs \
    librocksdb-dev libfuse2 \
    libpython3-dev \
    pcscd \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

USER jenkins

LABEL maintainer="marko@status.im"
LABEL source="https://github.com/status-im/status-desktop"
LABEL description="Build image for the Status Desktop e2e tests with Squish and Qt."