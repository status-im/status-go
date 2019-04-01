GIT_ROOT = $(shell git rev-parse --show-toplevel)

RED    := $(shell tput -Txterm setaf 1)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)
BOLD   := $(shell tput bold)

# Settings
export LOG_LEVEL      ?= INFO
export LISTEN_PORT    ?= 30303
export RPC_PORT       ?= 8545
export API_MODULES    ?= eth,net,web3,admin
export CONTAINER_NAME ?= status-go-mailserver
export DATA_PATH      ?= /var/tmp/status-go-mail
export FLEET_NAME     ?= eth.beta
export REGISTER_TOPIC ?= whispermail
export MAIL_PASSWORD  ?= status-offline-inbox
# Necessary to make mailserver available publicly
export PUBLIC_IP      ?= $(shell curl -s https://ipecho.net/plain)

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

start: config
	docker-compose up -d

stop:
	docker-compose down

logs:
	docker-compose logs -f -t --tail=100

enode:
	@curl -s -XPOST http://localhost:$(RPC_PORT)/ \
		-H 'Content-type: application/json' \
		-d '{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":1}' \
		| jq -r '.result.enode' \
		| grep -oP '\Kenode://[^?]+'


info:
	@echo "$(GREEN)Your mailserver is listening on:$(RESET) $(BOLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RESET)"
	@echo "$(YELLOW)Make sure that address and TCP port are available from the internet!$(RESET)"
	@echo "$(GREEN)Your enode address is:$(RESET)"

show:
	@docker ps --filter='name=$(CONTAINER_NAME)' --format="table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

config: ##@ Generate config for mailserver with current public IP.
	@cat $(GIT_ROOT)/config/cli/fleet-$(FLEET_NAME).json \
		| jq '.AdvertiseAddr = "$(PUBLIC_IP)"' \
		| jq '.HTTPEnabled = true' \
		| jq '.HTTPHost = "0.0.0.0"' \
		| jq '.HTTPPort= $(RPC_PORT)' \
		| jq '.APIModules = "$(API_MODULES)"' \
		| jq '.RegisterTopics = ["$(REGISTER_TOPIC)"]' \
		| jq '.WhisperConfig.Enabled = true' \
		| jq '.WhisperConfig.EnableMailServer = true' \
		| jq '.WhisperConfig.LightClient = false' \
		| jq '.WhisperConfig.MailServerPassword = "$(MAIL_PASSWORD)"' \
		> config.json

clean:
	rm -f config.json
	docker-compose rm -s -f
