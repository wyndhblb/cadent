# Makefile for a go project
#
# modified from: http://joneisen.me/post/25503842796
# 	
# Targets:
# 	all: Builds the code
# 	build: Builds the code
# 	fmt: Formats the source files
# 	clean: cleans the code
# 	install: Installs the code to the GOPATH
# 	iref: Installs referenced projects
#	test: Runs the tests
#
 
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOINSTALL=$(GOCMD) install
GOTEST=$(GOCMD) test
GODEP=$(GOTEST) -i
GOFMT=gofmt -w -s
 
# Package lists
TOPLEVEL_PKG := .
INT_LIST := 	#<-- Interface directories
IMPL_LIST := src/cadent/server	#<-- Implementation directories
CMD_LIST := src/cadent/cmd/cadent src/cadent/cmd/echoserver src/cadent/cmd/statblast src/cadent/cmd/readblast	#<-- Command directories
 
# List building
ALL_LIST = $(INT_LIST) $(IMPL_LIST) $(CMD_LIST)
 
BUILD_LIST = $(foreach int, $(ALL_LIST), $(int)_build)
CLEAN_LIST = $(foreach int, $(ALL_LIST), $(int)_clean)
INSTALL_LIST = $(foreach int, $(ALL_LIST), $(int)_install)
IREF_LIST = $(foreach int, $(ALL_LIST), $(int)_iref)
TEST_LIST = $(foreach int, $(ALL_LIST), $(int)_test)
FMT_TEST = $(foreach int, $(ALL_LIST), $(int)_fmt)
 
# All are .PHONY for now because dependencyness is hard
.PHONY: $(CLEAN_LIST) $(TEST_LIST) $(FMT_LIST) $(INSTALL_LIST) $(BUILD_LIST) $(IREF_LIST)
SHELL     := /bin/bash
all: build
build: $(BUILD_LIST)
cmd: $(CMD_LIST)
clean: $(CLEAN_LIST)
install: $(INSTALL_LIST)
test: $(TEST_LIST)
iref: $(IREF_LIST)
fmt: $(FMT_TEST)
export GOPATH=$(shell pwd)
export GOEXPERIMENT=noframepointer

#if [ which godep ]; then; \
#export GOPATH=$GOPATH:$(shell godep path ) \
#fi \

export build=$(shell git rev-parse HEAD)
export version=$(shell cat ./version )
export dated=$(shell date  +'%Y.%m.%d-%H.%M.%S')

$(BUILD_LIST): %_build: %_fmt %_iref
	$(GOBUILD) -v -ldflags "-X main.CadentBuild='$(version)-$(build)-$(dated)'" $(TOPLEVEL_PKG)/$*

$(CLEAN_LIST): %_clean:
	rm -f echoserver
	rm -f consthash
	rm -f statblast
	rm -f readblast
	$(GOCLEAN) $(TOPLEVEL_PKG)/$*
	
$(INSTALL_LIST): %_install:
	$(GOINSTALL) $(TOPLEVEL_PKG)/$*
	
$(IREF_LIST): %_iref:
	$(GODEP) $(TOPLEVEL_PKG)/$*
	
$(TEST_LIST): %_test:
	$(GOTEST) $(TOPLEVEL_PKG)/$*
	
$(FMT_TEST): %_fmt:
	$(GOFMT) ./$*