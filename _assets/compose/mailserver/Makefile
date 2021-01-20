export GIT_ROOT = $(shell git rev-parse --show-toplevel)

RED := $(shell tput -Txterm setaf 1)
GRN := $(shell tput -Txterm setaf 2)
YLW := $(shell tput -Txterm setaf 3)
RST := $(shell tput -Txterm sgr0)
BLD := $(shell tput bold)

# Settings
export CONTAINER_TAG  ?= v0.64.3
export CONTAINER_IMG  ?= statusteam/status-go
export CONTAINER_NAME ?= status-go-mailserver
export LOG_LEVEL      ?= INFO
export LISTEN_PORT    ?= 30303
export METRICS_PORT   ?= 9090
export RPC_PORT       ?= 8545
export API_MODULES    ?= eth,net,web3,admin,mailserver
export DATA_PATH      ?= /var/tmp/status-go-mail
export REGISTER_TOPIC ?= whispermail
export MAIL_PASSWORD  ?= status-offline-inbox
# Necessary to make mailserver available publicly
export PUBLIC_IP      ?= $(shell curl -s https://ipecho.net/plain)

all: checks start show info enode enode-qr

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
ifndef PUBLIC_IP
	$(error $(RED)$$PUBLIC_IP not set! Export it as environment variable$(RST))
endif
ifndef CONTAINER_NAME
	$(error $(RED)$$CONTAINER_NAME not set! Export it as environment variable$(RST))
endif

start: config
	docker-compose up -d

stop:
	docker-compose down

logs:
	docker-compose logs -f -t --tail=100

enode:
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh

enode-qr:
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh --qr

config:
	@$(GIT_ROOT)/_assets/scripts/gen_config.sh

info:
	@echo "$(GRN)Your mailserver is listening on:$(RST) $(BLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RST)"
	@echo "$(YLW)Make sure that address and TCP port are available from the internet!$(RST)"
	@echo "$(GRN)Your enode address is:$(RST)"

show:
	@docker ps \
		--filter='name=$(CONTAINER_NAME)' \
		--format="table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

clean:
	docker-compose rm -s -f
