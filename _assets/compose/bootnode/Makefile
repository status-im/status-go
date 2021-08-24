export GIT_ROOT = $(shell git rev-parse --show-toplevel)

RED := $(shell tput -Txterm setaf 1)
GRN := $(shell tput -Txterm setaf 2)
YLW := $(shell tput -Txterm setaf 3)
RST := $(shell tput -Txterm sgr0)
BLD := $(shell tput bold)

UID = $(shell id -u)
GID = $(shell id -g)

# Settings
export CONTAINER_TAG  ?= v0.64.3
export CONTAINER_IMG  ?= statusteam/bootnode
export CONTAINER_NAME ?= status-go-bootnode
export LOG_LEVEL      ?= 3
export LISTEN_PORT    ?= 30301
export API_MODULES    ?= eth,web3,admin
# Useful for showing enode address
export PUBLIC_IP ?= $(shell curl -s https://ipecho.net/plain)

NODE_ADDR  = $(shell cat keys/nodeaddr)
ENODE_ADDR = enode://$(NODE_ADDR)@$(PUBLIC_IP):$(LISTEN_PORT)

define INFO_MSG
 * $(GRN)Your bootnode is listening on:$(RST) $(BLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RST)
 * $(GRN)Your enode address is:$(RST)
$(ENODE_ADDR)

$(YLW)Make sure that address and UDP port are available from the internet!$(RST)

endef
export INFO_MSG

all: checks start show info enode-qr

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

enode: keys/nodeaddr
	@echo $(ENODE_ADDR)

enode-qr: keys/nodeaddr
	@qrencode -t UTF8 $(ENODE_ADDR)

logs: LOG_LINES ?= 100
logs:
	docker-compose logs -f -t --tail=$(LOG_LINES)

info:
	@echo "$$INFO_MSG"

keys:
	@mkdir -p keys

start: keys/nodekey keys/nodeaddr
	@echo " * $(GRN)Starting '$(CONTAINER_NAME)' container...$(RST)"
	docker-compose up -d

stop:
	@echo " * $(YLW)Stopping '$(CONTAINER_NAME)' container...$(RST)"
	docker-compose down

keys/nodekey: keys ##@ Generate a node key
	@echo " * $(GRN)Generating '$(CONTAINER_NAME)' keys...$(RST)"
	@docker run --rm \
		-u $(UID):$(GID) \
		--entrypoint=bootnode \
		-v $(PWD)/keys:/keys:rw \
		$(CONTAINER_IMG) \
		-genkey=/keys/nodekey
	@echo " * $(GRN)Created key for Bootnode: keys/nodekey$(RST)"

keys/nodeaddr: keys ##@ Save node address for given key
	@echo " * $(GRN)Saving '$(CONTAINER_NAME)' enode address...$(RST)"
	@docker run --rm \
		-u $(UID):$(GID) \
		--entrypoint=sh \
		-v $(PWD)/keys:/keys:rw \
		$(CONTAINER_IMG) \
		-c 'bootnode -writeaddress -nodekey=/keys/nodekey > /keys/nodeaddr'

show:
	@docker ps --filter='name=$(CONTAINER_NAME)' --format="table {{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"

clean:
	docker-compose rm -s -f
