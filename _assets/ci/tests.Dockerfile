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

USER jenkins

ENTRYPOINT [""]
