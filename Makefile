.PHONY: all
all: nexlink

.PHONY: nexlink
nexlink: dist/nexlink

dist/nexlink: cmd/nexlink/main.go
	go build -o $@ $<
