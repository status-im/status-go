export GIT_ROOT := $(shell git rev-parse --show-toplevel)
# Useful for showing enode address
PUBLIC_IP ?= $(shell curl -s https://ipecho.net/plain)

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
export SERVICE_NAME ?= status-go-mailserver
export SERVICE_DIR  ?= $(HOME)/.config/systemd/user/
export SERVICE_PATH ?= $(SERVICE_DIR)/$(SERVICE_NAME).service
export DATA_PATH    ?= /var/tmp/$(SERVICE_NAME)
export LOG_LEVEL    ?= INFO
export LISTEN_PORT  ?= 30303
export METRICS_PORT ?= 9090
export RPC_PORT     ?= 8545

# Info
STATUS = $(shell systemctl $(SCTL_OPTS) is-active $(SERVICE_NAME))

define INFO_MSG
 * $(GRN)Your mailserver is listening on:$(RST) $(BLD)$(PUBLIC_IP):$(LISTEN_PORT)$(RST)

$(GRN)Your enode address is:$(RST)
$(shell $(GIT_ROOT)/_assets/scripts/get_enode.sh 2>/dev/null)

$(YLW)Make sure that IP and TCP port are available from the internet!$(RST)

endef
export INFO_MSG

all: checks build config service enable restart info enode-qr

clean: stop disable rm-service forget 

checks:
ifeq (, $(shell which jq))
	$(error $(RED)No 'jq' in your $$PATH. please install it$(RST))
endif
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
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh

enode-qr:
	@$(GIT_ROOT)/_assets/scripts/get_enode.sh --qr

status:
	systemctl $(SCTL_OPTS) status --no-pager $(SERVICE_NAME)

logs:
	journalctl $(JRNL_OPTS) statusd

enable:
	@echo " * $(GRN)Enabling '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) enable $(SERVICE_NAME)

disable:
	@echo " * $(YLW)Disabling '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) disable "${SERVICE_NAME}"

start:
	@echo " * $(GRN)Starting '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) start $(SERVICE_NAME)

restart:
	@echo " * $(GRN)Restarting '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) restart $(SERVICE_NAME)

stop:
	@echo " * $(YLW)Stopping '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) stop "${SERVICE_NAME}"

forget:
	@echo " * $(YLW)Stopping '$(SERVICE_NAME)' service...$(RST)"
	systemctl $(SCTL_OPTS) daemon-reload
	systemctl $(SCTL_OPTS) reset-failed

$(GIT_ROOT)/build/bin/statusd:
	@echo " * $(GRN)Building mailserver binary...$(RST)"
	@cd "$(GIT_ROOT)" && make statusgo

build: $(GIT_ROOT)/build/bin/statusd
.PHONY: build

$(SERVICE_DIR):
	@mkdir -p $(SERVICE_DIR)

service: $(SERVICE_DIR)
	@echo " * $(GRN)Generating '$(SERVICE_NAME)' service...$(RST)"
	@envsubst < ./service.template > $(SERVICE_PATH)

rm-service:
	rm -f $(SERVICE_PATH)

config:
	@echo " * $(GRN)Generating '$(SERVICE_NAME)' config...$(RST)"
	@$(GIT_ROOT)/_assets/scripts/gen_config.sh
