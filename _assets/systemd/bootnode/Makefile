export GIT_ROOT = $(shell git rev-parse --show-toplevel)

RED := $(shell tput -Txterm setaf 1)
GRN := $(shell tput -Txterm setaf 2)
YLW := $(shell tput -Txterm setaf 3)
RST := $(shell tput -Txterm sgr0)
BLD := $(shell tput bold)

# by default we run the service as user
SCTL_OPTS   := --user
JRNL_OPTS   := --user-unit
SERVICE_DIR := $(HOME)/.config/systemd/user/
# with sudo it will run as a system service
ifeq ($(USER), root)
SCTL_OPTS   := 
JRNL_OPTS   := --unit
SERVICE_DIR := /etc/systemd/system/
endif

# Settings
export SERVICE_NAME ?= bootnode
export SERVICE_PATH ?= $(SERVICE_DIR)/$(SERVICE_NAME).service
export LOG_LEVEL    ?= 3
export LISTEN_PORT  ?= 30303
export DATA_PATH    ?= /var/tmp/status-go-boot
export KEY_PATH     ?= $(DATA_PATH)/nodekey
export ADDR_PATH    ?= $(DATA_PATH)/nodeaddr
# Necessary to make bootnode available publicly
export PUBLIC_IP    ?= $(shell curl -s https://ipecho.net/plain)

# Info
STATUS    = $(shell systemctl $(SCTL_OPTS) is-active $(SERVICE_NAME))
NODE_ADDR = $(shell cat $(ADDR_PATH))
ENODE     = enode://$(NODE_ADDR)@$(PUBLIC_IP):$(LISTEN_PORT)

define INFO_MSG
* $(GRN)Your bootnode is listening on:$(RST) $(BLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RST)
* $(YLW)Make sure that IP and TCP port are available from the internet!$(RST)
$(GRN)Your enode address is:$(RST)
$(ENODE)
endef
export INFO_MSG

all: checks build genkey save-address service enable restart info

clean: stop disable rm-service forget 

checks:
# this setup wont work without an os with systemd
ifeq (, $(shell which systemctl))
	$(error $(RED)Your system does not have systemd$(RST))
endif
# if the service is already up just show some info
ifeq (active, $(STATUS))
	$(info $(INFO_MSG))
	$(error $(YLW)Service already started$(RST))
endif

info:
	@echo "$$INFO_MSG"

enode:
	@echo "$(ENODE)"

genkey: $(DATA_PATH)
	@$(GIT_ROOT)/build/bin/bootnode -genkey=$(KEY_PATH)

address: save-address
	echo "$(NODE_ADDR)"

save-address:
	@$(GIT_ROOT)/build/bin/bootnode -nodekey=$(KEY_PATH) -writeaddress > $(ADDR_PATH)

status:
	systemctl $(SCTL_OPTS) status --no-pager $(SERVICE_NAME)

logs:
	journalctl $(JRNL_OPTS) statusd

enable:
	@echo "* $(BLD)Enabling '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) enable $(SERVICE_NAME)

disable:
	@echo "* $(BLD)Disabling '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) disable "${SERVICE_NAME}"

start:
	@echo "* $(BLD)Starting '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) start $(SERVICE_NAME)

restart:
	@echo "* $(BLD)Restarting '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) restart $(SERVICE_NAME)

stop:
	@echo "* $(BLD)Stopping '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) stop "${SERVICE_NAME}"

forget:
	@echo "* $(BLD)Stopping '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) daemon-reload
	systemctl $(SCTL_OPTS) reset-failed

$(GIT_ROOT)/build/bin/bootnode:
	@echo "* $(BLD)Building bootnode binary...$(RST)"
	@cd "$(GIT_ROOT)" && make bootnode

build: $(GIT_ROOT)/build/bin/bootnode
.PHONY: build

$(SERVICE_DIR):
	echo $(SERVICE_DIR)
	@mkdir -p $(SERVICE_DIR)

service: $(SERVICE_DIR)
	@echo "* $(BLD)Generating '$(SERVICE_NAME)' service...$(RST)"
	@envsubst < ./service.template > $(SERVICE_PATH)

$(DATA_PATH):
	@mkdir -p $(DATA_PATH)

rm-service:
	rm -f $(SERVICE_PATH)
