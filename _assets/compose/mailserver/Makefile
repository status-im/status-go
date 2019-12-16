export GIT_ROOT = $(shell git rev-parse --show-toplevel)

RED    := $(shell tput -Txterm setaf 1)
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)
BOLD   := $(shell tput bold)

# Settings
export LOG_LEVEL      ?= INFO
export LISTEN_PORT    ?= 30303
export RPC_PORT       ?= 8545
export API_MODULES    ?= eth,net,web3,admin,mailserver
export CONTAINER_NAME ?= status-go-mailserver
export DATA_PATH      ?= /var/tmp/status-go-mail
export FLEET_NAME     ?= eth.beta
export REGISTER_TOPIC ?= whispermail
export MAIL_PASSWORD  ?= status-offline-inbox
# Necessary to make mailserver available publicly
export PUBLIC_IP      ?= $(shell curl -s https://ipecho.net/plain)

all: checks start show info enode

checks:
ifeq (, $(shell which docker))
	$(error No $(BOLD)docker$(RESET) in your $$PATH. Please install it)
endif
ifeq (, $(shell docker version | grep Server))
	$(error No permissions to run $(BOLD)docker$(RESET). Add yourself to docker group)
endif
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
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh

config:
	@$(GIT_ROOT)/_assets/scripts/gen_config.sh

info:
	@echo "$(GREEN)Your mailserver is listening on:$(RESET) $(BOLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RESET)"
	@echo "$(YELLOW)Make sure that address and TCP port are available from the internet!$(RESET)"
	@echo "$(GREEN)Your enode address is:$(RESET)"

show:
	@docker ps \
		--filter='name=$(CONTAINER_NAME)' \
		--format="table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

clean:
	docker-compose rm -s -f
