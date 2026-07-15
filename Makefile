# crc 개발 태스크. 훅(lefthook)·로컬이 같은 타깃을 공유한다.
.PHONY: setup fmt lint test build check install

PREFIX ?= $(HOME)/.local/bin

setup:   ; lefthook install
fmt:     ; golangci-lint fmt ./...
lint:    ; golangci-lint run ./...
test:    ; go test ./...
build:   ; go build -o crc .
check: lint test build
# ~/.local/bin(PREFIX로 변경 가능)에 crc를 설치한다.
install: ; go build -o "$(PREFIX)/crc" . && echo "설치: $(PREFIX)/crc"
