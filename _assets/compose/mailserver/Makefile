export GIT_ROOT = $(shell git rev-parse --show-toplevel)
# Useful for showing enode address
PUBLIC_IP ?= $(shell curl -s https://ipecho.net/plain)

RED := $(shell tput -Txterm setaf 1)
GRN := $(shell tput -Txterm setaf 2)
YLW := $(shell tput -Txterm setaf 3)
RST := $(shell tput -Txterm sgr0)
BLD := $(shell tput bold)

# Settings
export CONTAINER_TAG  ?= v0.84.0
export CONTAINER_IMG  ?= statusteam/status-go
export CONTAINER_NAME ?= status-go-mailserver
export LOG_LEVEL      ?= INFO
export LISTEN_PORT    ?= 30303
export METRICS_PORT   ?= 9090
export RPC_HOST       ?= 0.0.0.0
export RPC_PORT       ?= 8545
export API_MODULES    ?= eth,web3,admin,waku,wakuext
export DATA_PATH      ?= /var/tmp/$(CONTAINER_NAME)
export REGISTER_TOPIC ?= whispermail
export MAIL_PASSWORD  ?= status-offline-inbox

define INFO_MSG
 * $(GRN)Your mailserver is listening on:$(RST) $(BLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RST)
 * $(GRN)Your enode address is:$(RST)
$(shell $(GIT_ROOT)/_assets/scripts/get_enode.sh 2>/dev/null)

$(YLW)Make sure that IP and TCP port are available from the internet!$(RST)

endef
export INFO_MSG

all: checks start show info enode-qr

status: show

checks:
ifeq (, $(shell which docker))
	$(error $(RED)No 'docker' in your $$PATH. Please install it$(RST))
endif
ifeq (, $(shell docker version | grep Server))
	$(error $(RED)No permissions to run 'docker'. Add yourself to docker group$(RST))
endif
ifeq (, $(shell which docker-compose))
	$(error $(RED)No 'docker-compose' in your $$PATH. Please install it$(RST))
endif
ifeq (, $(shell which jq))
	$(error $(RED)No 'jq' in your $$PATH. Please install it$(RST))
endif
ifndef CONTAINER_NAME
	$(error $(RED)$$CONTAINER_NAME not set! Export it as environment variable$(RST))
endif
ifndef PUBLIC_IP
	$(warning $(YLW)$$PUBLIC_IP not set! Export it as environment variable$(RST))
endif

info:
	@echo "$$INFO_MSG"

enode:
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh

enode-qr:
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh --qr

logs: LOG_LINES ?= 100
logs:
	docker-compose logs -f -t --tail=$(LOG_LINES)

start: config
	@echo " * $(GRN)Starting '$(CONTAINER_NAME)' container...$(RST)"
	docker-compose $(COMPOSE_UP_FLAGS) up -d

stop:
	@echo " * $(YLW)Stopping '$(CONTAINER_NAME)' container...$(RST)"
	docker-compose down

config:
	@echo " * $(GRN)Generating '$(CONTAINER_NAME)' config...$(RST)"
	@$(GIT_ROOT)/_assets/scripts/gen_config.sh

show:
	@docker ps \
		--filter='name=$(CONTAINER_NAME)' \
		--format="table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

clean:
	@echo " * $(YLW)Removing '$(CONTAINER_NAME)' container...$(RST)"
	docker-compose rm -s -f
