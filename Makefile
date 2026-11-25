BIN ?= $(HOME)/.local/bin/ghm

.PHONY: build app icon test vet check install doctor fmt clean

build:
	go build -o $(BIN) ./cmd/ghm

app: install
	GHM_SRC=$(BIN) ./app/scripts/bundle.sh release

icon:
	cd app && rm -rf build/AppIcon.iconset && swift scripts/make-icon.swift build/AppIcon.iconset && iconutil -c icns build/AppIcon.iconset -o Resources/AppIcon.icns

install: build
	@echo "installed $(BIN) - run 'ghm install-shim' once per machine"

test:
	go test ./...

vet:
	go vet ./...

# Everything the (disabled) CI job used to run, but on this machine.
check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt: run 'make fmt'"; exit 1; }
	go vet ./...
	go test ./...
	go build ./...

fmt:
	gofmt -l -w .

doctor: install
	$(BIN) doctor

clean:
	rm -f ghm
