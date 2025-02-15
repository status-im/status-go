FROM ghcr.io/foundry-rs/foundry:stable

USER root

RUN apt-get update && \
    apt-get install -y git bash

WORKDIR /app

COPY entrypoint.sh /app
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT [ "/app/entrypoint.sh" ]
WORKDIR /app/$GITHUB_REPO
