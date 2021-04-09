SHORT_PACKAGE=rmproxy
PACKAGE=github.com/kwkoo/broadlinkrm

BASE:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
COVERAGEOUTPUT=coverage.out
COVERAGEHTML=coverage.html
IMAGENAME="kwkoo/golang_broadlink_rm"
VERSION="0.1"

.PHONY: build clean test coverage run image runcontainer
build:
	@echo "Building..."
	@cd $(BASE)/src && go build -o $(BASE)/bin/$(SHORT_PACKAGE) \
		$(PACKAGE)/cmd/$(SHORT_PACKAGE)

clean:
	rm -f $(BASE)/bin/$(SHORT_PACKAGE)

rundemo:
	@cd $(BASE)/src && go run cmd/demo/main.go

runrmproxy:
	@cd $(BASE)/src && go run \
		cmd/$(SHORT_PACKAGE)/main.go \
		-key 123 \
		-rooms $(BASE)/../localremote/json/rooms.json \
		-commands $(BASE)/../localremote/json/commands.json \
		-macros $(BASE)/../localremote/json/macros.json \
		-homeassistant $(BASE)/../localremote/json/homeassistant.json \
		-deviceconfig $(BASE)/../localremote/json/devices.json \
		-skipdiscovery

runmacrobuilder:
	@cd $(BASE)/src && go run \
			cmd/macrobuilder/main.go \
			-rooms $(BASE)/../localremote/json/rooms.json \
			-commands $(BASE)/../localremote/json/commands.json

image: 
	docker build --rm -t $(IMAGENAME):$(VERSION) $(BASE)

runcontainer:
	docker run \
		--rm \
		-it \
		--env-file $(BASE)/env.list \
		--name $(SHORT_PACKAGE) \
		-p 8080:8080 \
		$(IMAGENAME):$(VERSION) \