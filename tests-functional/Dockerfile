FROM ghcr.io/foundry-rs/foundry:v0.3.0

RUN apk update && \
    apk add git bash

WORKDIR /app

COPY clone_and_run.sh /app
RUN chmod +x /app/clone_and_run.sh

COPY entrypoint.sh /app
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT [ "/app/entrypoint.sh" ]