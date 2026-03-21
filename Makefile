VERSION ?= dev

build:
	go build -ldflags "-X github.com/eduardmaghakyan/why/cmd.Version=$(VERSION)" -o why .

test:
	go test ./...

install: build
	mkdir -p $(HOME)/.why/bin
	cp why $(HOME)/.why/bin/why
	@echo "Installed to $(HOME)/.why/bin/why"
	@echo 'Add to PATH: export PATH="$$HOME/.why/bin:$$PATH"'

clean:
	rm -f why

.PHONY: build test install clean
