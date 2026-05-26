BIN_DIR := bin
BINARY := $(BIN_DIR)/pswitch
MODE ?= round_robin
LISTEN ?= 0.0.0.0:8080
LOG_COLOR ?=
GO ?= go

.PHONY: help build run test clean

help:
	@printf '%s\n' \
		'Targets:' \
		'  make build            Build the pswitch binary into ./bin/pswitch' \
		'  make run              Run pswitch with LISTEN=$(LISTEN), MODE=$(MODE), and optional LOG_COLOR=$(LOG_COLOR)' \
		'  make test             Run go test ./...' \
		'  make clean            Remove the build output'

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) ./cmd/pswitch

run: build
	@if [ -n "$(LOG_COLOR)" ]; then \
		$(BINARY) --listen $(LISTEN) --mode $(MODE) --log-color=$(LOG_COLOR); \
	else \
		$(BINARY) --listen $(LISTEN) --mode $(MODE); \
	fi

test:
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
