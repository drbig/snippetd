PROJECT := $(shell basename `pwd`)
TGTS    := amd64 rpi1
VER     := $(shell grep -Eo 'VERSION = `(.*)`' main.go | cut -d'`' -f2)
BUILD   := $(shell echo `whoami`@`hostname -s` on `date`)
LDFLAGS := -ldflags='-X "main.build=$(BUILD)"'

.PHONY: clean all test

all: $(TGTS) bin/checksums.md5

test: $(TGTS)

clean:
	@rm -f bin/*

dev:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./$(PROJECT)-$@-$(VER)-dev main.go

amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(PROJECT)-$@-$(VER) main.go

rpi1:
	GOOS=linux GOARCH=arm GOARM=5 go build $(LDFLAGS) -o bin/$(PROJECT)-$@-$(VER) main.go

bin/checksums.md5:
	cd bin && md5sum * > checksums.md5
