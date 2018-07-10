SHORT_PACKAGE=rmproxy
PACKAGE=github.com/kwkoo/broadlinkrm

GOPATH:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GOBIN=$(GOPATH)/bin
COVERAGEOUTPUT=coverage.out
COVERAGEHTML=coverage.html
IMAGENAME="kwkoo/golang_broadlink_rm"
VERSION="0.1"

.PHONY: build clean test coverage run image runcontainer
build:
	@echo "Building..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/$(SHORT_PACKAGE) $(PACKAGE)/cmd/$(SHORT_PACKAGE)

clean:
	rm -f $(GOPATH)/bin/$(SHORT_PACKAGE) $(GOPATH)/pkg/*/$(PACKAGE).a $(GOPATH)/$(COVERAGEOUTPUT) $(GOPATH)/$(COVERAGEHTML)

test:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go test $(PACKAGE)

coverage:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go test $(PACKAGE) -cover -coverprofile=$(GOPATH)/$(COVERAGEOUTPUT)
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go tool cover -html=$(GOPATH)/$(COVERAGEOUTPUT) -o $(GOPATH)/$(COVERAGEHTML)
	open $(GOPATH)/$(COVERAGEHTML)

run:
	@GOPATH=$(GOPATH) go run $(GOPATH)/src/$(PACKAGE)/cmd/$(SHORT_PACKAGE)/main.go -key 123 -rooms $(GOPATH)/json/rooms.json -commands $(GOPATH)/json/commands.json

image: 
	docker build --rm -t $(IMAGENAME):$(VERSION) $(GOPATH)

runcontainer:
	docker run --rm -it --name $(SHORT_PACKAGE) -p 8080:8080 $(IMAGENAME):$(VERSION)
