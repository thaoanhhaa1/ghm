BIN ?= $(HOME)/.local/bin/ghm

.PHONY: build app test vet install doctor fmt clean

build:
	go build -o $(BIN) ./cmd/ghm

app: install
	GHM_SRC=$(BIN) ./app/scripts/bundle.sh release

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
