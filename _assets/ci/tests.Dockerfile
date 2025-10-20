FROM harbor.status.im/infra/ci-build-containers:linux-base-1.0.0

USER root

RUN apt-get update && apt-get install -yq --no-install-recommends --fix-missing \
    lsb-release \
    xz-utils \
    gnupg \
    wget \
    build-essential \
    python3 \
    python3-pip \
    python3-venv \
    openssh-client \
    && rm -rf /var/lib/apt/lists/*

RUN ln -sf /usr/bin/python3 /usr/bin/python

RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/debian/gpg  | gpg --batch --yes --dearmor -o /etc/apt/keyrings/docker.gpg && \
    gpg --no-default-keyring --keyring /etc/apt/keyrings/docker.gpg --fingerprint | \
    grep -q "8D81 803C 0EBF CD88" && \
    echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/debian  \
    bullseye stable" \
    | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && apt-get install -y \
    docker-ce-cli \
    docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

USER jenkins

ENTRYPOINT [""]
