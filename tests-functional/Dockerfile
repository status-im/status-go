FROM ghcr.io/foundry-rs/foundry:latest

RUN apk update && \
    apk add git bash

WORKDIR /app

COPY entrypoint.sh /app
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT [ "/app/entrypoint.sh" ]
WORKDIR /app/$GITHUB_REPO
