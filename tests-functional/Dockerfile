FROM ghcr.io/foundry-rs/foundry:v0.3.0

RUN apk update && \
    apk add git bash

WORKDIR /app

COPY entrypoint.sh /app
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT [ "/app/entrypoint.sh" ]
WORKDIR /app/$GITHUB_REPO
