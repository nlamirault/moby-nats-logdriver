# Copyright (C) 2017 Nicolas Lamirault <nicolas.lamirault@gmail.com>

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writingmoby-nats-logdriver, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

APP = moby-nats-logdriver

VERSION=$(shell \
        grep "const Version" version/version.go \
        |awk -F'=' '{print $$2}' \
        |sed -e "s/[^0-9.]//g" \
	|sed -e "s/ //g")

SHELL = /bin/bash

DIR = $(shell pwd)

DOCKER = docker
NAMESPACE = nlamirault

GO = go

GOX = gox -os="linux darwin windows"
GOX_ARGS = "-output={{.Dir}}-$(VERSION)_{{.OS}}_{{.Arch}}"

BINTRAY_URI = https://api.bintray.com
BINTRAY_USERNAME = nlamirault
BINTRAY_ORG = nlamirault
BINTRAY_REPOSITORY= oss

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

MAKE_COLOR=\033[33;01m%-20s\033[0m

SRCS = $(shell git ls-files '*.go' | grep -v '^vendor/')
EXE = $(shell ls moby-nats-logdriver-*_*)

PACKAGE=$(APP)-$(VERSION)
ARCHIVE=$(PACKAGE).tar
PKGS = $(shell go list ./... | grep -v /vendor/)

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo -e "$(OK_COLOR)==== $(APP) [$(VERSION)] ====$(NO_COLOR)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(MAKE_COLOR) : %s\n", $$1, $$2}'

clean: ## Cleanup
	@echo -e "$(OK_COLOR)[$(APP)] Cleanup$(NO_COLOR)"
	@rm -fr $(APP) $(EXE) $(APP)-*.tar.gz

.PHONY: init
init: ## Install requirements
	@echo -e "$(OK_COLOR)[$(APP)] Install requirements$(NO_COLOR)"
	@go get -u github.com/golang/glog
	@go get -u github.com/kardianos/govendor
	@go get -u github.com/Masterminds/rmvcsdir
	@go get -u github.com/golang/lint/golint
	@go get -u github.com/kisielk/errcheck
	@go get -u github.com/mitchellh/gox

.PHONY: deps
deps: ## Install dependencies
	@echo -e "$(OK_COLOR)[$(APP)] Update dependencies$(NO_COLOR)"
	@govendor update

.PHONY: build
build: ## Make binary
	@echo -e "$(OK_COLOR)[$(APP)] Build $(NO_COLOR)"
	@$(GO) build .

.PHONY: test
test: ## Launch unit tests
	@echo -e "$(OK_COLOR)[$(APP)] Launch unit tests $(NO_COLOR)"
	@govendor test +local

.PHONY: lint
lint: ## Launch golint
	@$(foreach file,$(SRCS),golint $(file) || exit;)

.PHONY: vet
vet: ## Launch go vet
	@$(foreach file,$(SRCS),$(GO) vet $(file) || exit;)

.PHONY: errcheck
errcheck: ## Launch go errcheck
	@echo -e "$(OK_COLOR)[$(APP)] Go Errcheck $(NO_COLOR)"
	@$(foreach pkg,$(PKGS),errcheck $(pkg) || exit;)

.PHONY: coverage
coverage: ## Launch code coverage
	@$(foreach pkg,$(PKGS),$(GO) test -cover $(pkg) || exit;)

plugin-package:
	@set -e ;\
		rm -fr rootfs ;\
		docker build -t $(NAMESPACE)/$(APP):$(VERSION) . ;\
		id=$$(docker create $(NAMESPACE)/$(APP):$(VERSION) true) ;\
		echo $$id ;\
		mkdir -p plugin/rootfs ;\
		docker export $$id | tar -x -C plugin/rootfs/ ;\
		docker rm -vf $$id ;\
		docker rmi $(NAMESPACE)/$(APP):$(VERSION) ;\
		cp config.json plugin ;\

.PHONY: plugin-install
plugin-install: plugin-package ## Install the Docker plugin
	@echo -e "$(OK_COLOR)[$(APP)] Plugin install $(APP):$(VERSION) $(NO_COLOR)"
	@$(DOCKER) plugin create $(NAMESPACE)/$(APP):$(VERSION) plugin
	@echo "Now configure the Nats server with 'docker plugin set $(NAMESPACE)/$(APP):$(VERSION) NATS_ADDRESS=nats://xxxxxx:4222'"
	@echo "Once configured, run 'make enable' to enable the plugin"

.PHONY: plugin-enable
plugin-enable: ## Enable the Docker plugin
	@echo -e "$(OK_COLOR)[$(APP)] Plugin enable $(APP):$(VERSION) $(NO_COLOR)"
	@$(DOCKER) plugin enable $(NAMESPACE)/$(APP):$(VERSION)

.PHONY: plugin-uninstall
plugin-uninstall: ## Uninstall the Docker plugin
	@echo -e "$(OK_COLOR)[$(APP)] Plugin uninstall $(APP):$(VERSION) $(NO_COLOR)"
	@$(DOCKER) plugin rm $(NAMESPACE)/$(APP):$(VERSION)

.PHONY: docker-build
docker-build: ## Build docker image
	@echo -e "$(OK_COLOR)[$(APP)] Build $(APP):$(VERSION) $(NO_COLOR)"
	@$(DOCKER) build -t $(NAMESPACE)/$(APP):$(VERSION) .

.PHONY: docker-publish
docker-publish: docker-build
	@echo -e "$(OK_COLOR)[$(APP)] Publish $(APP):$(VERSION) $(NO_COLOR)"
	@$(DOCKER) tag $(NAMESPACE)/$(APP):$(VERSION) $(NAMESPACE)/$(APP):$(VERSION)
	@$(DOCKER) push $(NAMESPACE)/$(APP):$(VERSION)

.PHONY: docker-run
docker-run: ## Run Blinky using image
	@$(DOCKER) run -it --rm=true --name $(NAMESPACE)/$(APP):$(VERSION) --help

gox: ## Make all binaries
	@echo -e "$(OK_COLOR)[$(APP)] Create binaries $(NO_COLOR)"
	$(GOX) $(GOX_ARGS) github.com/nlamirault/moby-nats-logdriver

.PHONY: binaries
binaries: ## Upload all binaries
	@echo -e "$(OK_COLOR)[$(APP)] Upload binaries to Bintray $(NO_COLOR)"
	for i in $(EXE); do \
		curl -T $$i \
			-u$(BINTRAY_USERNAME):$(BINTRAY_APIKEY) \
			"$(BINTRAY_URI)/content/$(BINTRAY_ORG)/$(BINTRAY_REPOSITORY)/$(APP)/${VERSION}/$$i;publish=1"; \
        done

# for goprojectile
.PHONY: gopath
gopath:
	@echo `pwd`:`pwd`/vendor
