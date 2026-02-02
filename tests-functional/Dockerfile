FROM ghcr.io/foundry-rs/foundry:v1.4.0

WORKDIR /app

# Copy contracts that need to be pre-deployed
RUN mkdir -p /app/contracts
COPY contracts/Multicall3.sol /app/contracts/

COPY --chmod=0755 clone_and_run.sh /app
COPY --chmod=0755 deploy_contracts.sh /app
COPY --chmod=0755 entrypoint.sh /app

ENTRYPOINT [ "/app/entrypoint.sh" ]