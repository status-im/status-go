FROM ghcr.io/foundry-rs/foundry:v1.3.0

WORKDIR /app

COPY --chmod=0755 clone_and_run.sh /app
COPY --chmod=0755 entrypoint.sh /app

ENTRYPOINT [ "/app/entrypoint.sh" ]