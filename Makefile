VERSION ?= dev

build:
	go build -ldflags "-X github.com/eduardmaghakyan/why/cmd.Version=$(VERSION)" -o why .

test:
	go test ./...

install: build
	cp why /usr/local/bin/why

clean:
	rm -f why

.PHONY: build test install clean
