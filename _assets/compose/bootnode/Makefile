GIT_ROOT = $(shell git rev-parse --show-toplevel)

RED    := $(shell tput -Txterm setaf 1)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)
BOLD   := $(shell tput bold)

UID = $(shell id -u)
GID = $(shell id -g)


# Settings
export LOG_LEVEL       ?= 3
export LISTEN_PORT     ?= 30303
export RPC_PORT        ?= 8545
export API_MODULES     ?= eth,net,web3,admin
export CONTAINER_NAME  ?= status-go-bootnode
export CONTAINER_IMAGE ?= statusteam/bootnode:latest
export TOOLS_IMAGE     ?= ethereum/client-go:alltools-latest
export FLEET_NAME      ?= eth.beta
# Necessary to make bootnode available publicly
export PUBLIC_IP       ?= $(shell curl -s https://ipecho.net/plain)

all: checks start show info enode

checks:
ifeq (, $(shell which docker-compose))
	$(error No $(BOLD)docker-compose$(RESET) in your $$PATH. Please install it)
endif
ifeq (, $(shell which jq))
	$(error No $(BOLD)jq$(RESET) in your $$PATH. Please install it)
endif
ifndef PUBLIC_IP
	$(error PUBLIC_IP not set! Export it as environment variable)
endif
ifndef CONTAINER_NAME
	$(error CONTAINER_NAME not set! Export it as environment variable)
endif

start: keys/nodekey keys/nodeaddr
	docker-compose up -d

stop:
	docker-compose down

logs:
	docker-compose logs -f -t --tail=100

enode: keys/nodeaddr
	@echo "enode://$(shell cat keys/nodeaddr)@$(PUBLIC_IP):$(LISTEN_PORT)"

info:
	@echo "$(GREEN)Your bootnode is listening on:$(RESET) $(BOLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RESET)"
	@echo "$(YELLOW)Make sure that address and UDP port are available from the internet!$(RESET)"
	@echo "$(GREEN)Your enode address is:$(RESET)"


keys:
	@mkdir -p keys

keys/nodekey: keys ##@ Generate a node key
	@docker run --rm \
		-u $(UID):$(GID) \
		--entrypoint=bootnode \
		-v $(PWD)/keys:/keys:rw \
		$(TOOLS_IMAGE) \
		-genkey=/keys/nodekey
	@echo "$(GREEN)Created key for Bootnode: keys/nodekey"

keys/nodeaddr: keys ##@ Save node address for given key
	@docker run --rm \
		-u $(UID):$(GID) \
		--entrypoint=sh \
		-v $(PWD)/keys:/keys:rw \
		$(TOOLS_IMAGE) \
		-c 'bootnode -writeaddress -nodekey=/keys/nodekey > /keys/nodeaddr'

show:
	@docker ps --filter='name=$(CONTAINER_NAME)' --format="table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

clean:
	docker-compose rm -s -f
