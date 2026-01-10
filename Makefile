.PHONY: build test clean

build:
	go build -o bin/cd-rip ./cmd/cd-rip
	go build -o bin/cd-encode ./cmd/cd-encode

test:
	go test ./...

clean:
	rm -f bin/cd-rip bin/cd-encode
