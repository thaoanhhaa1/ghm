BIN ?= $(HOME)/.local/bin/ghm

.PHONY: build test vet install doctor fmt clean

build:
	go build -o $(BIN) ./cmd/ghm

install: build
	@echo "installed $(BIN) - run 'ghm install-shim' once per machine"

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

doctor: install
	$(BIN) doctor

clean:
	rm -f ghm
