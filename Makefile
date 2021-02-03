VERSION := $(shell git describe)
LDFLAGS :=-ldflags "-X 'leto.LETO_VERSION=$(VERSION)'"

all: check-main leto/leto leto-cli/leto-cli

check-main:
	go test

leto/leto: *.go leto/*.go
	cd leto && go build $(LDFLAGS)

leto-cli/leto-cli: *.go leto-cli/*.go
	cd leto-cli && go build $(LDFLAGS)

clean:
	rm -f leto/leto
	rm -f leto-cli/leto-cli

.PHONY: clean
