FROM ghcr.io/foundry-rs/foundry:v1.4.0

WORKDIR /app

# Copy contracts that need to be pre-deployed
RUN mkdir -p /app/contracts
COPY contracts/Multicall3.sol /app/contracts/

COPY --chmod=0755 *.sh /app

ENTRYPOINT [ "/app/entrypoint.sh" ]