VER     := $(shell grep -Eo 'VERSION = `(.*)`' main.go | cut -d'`' -f2)
BUILD   := $(shell echo `whoami`@`hostname -s` on `date`)
LDFLAGS := -ldflags='-X "main.build=$(BUILD)"'

.PHONY: clean

clean:
	@rm -f snippetd-*

amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o snippetd-amd64-$(VER) .

rpi1:
	GOOS=linux GOARCH=arm GOARM=5 go build $(LDFLAGS) -o snippetd-rpi1-$(VER) .
