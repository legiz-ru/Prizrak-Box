# Makefile для сборки px-service

BINARY_NAME=px-service
GO=go

# Определение платформы
ifeq ($(OS),Windows_NT)
    BINARY_NAME := $(BINARY_NAME).exe
    RM := del /Q
else
    RM := rm -f
endif

# Цели сборки
.PHONY: all build clean deps windows linux darwin

all: build

deps:
	$(GO) mod download

build: deps
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BINARY_NAME) .

# Кросс-компиляция
windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags="-s -w" -o px-service.exe .

windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(GO) build -ldflags="-s -w" -o px-service.exe .

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o px-service .

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags="-s -w" -o px-service .

darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="-s -w" -o px-service .

darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="-s -w" -o px-service .

clean:
	$(RM) $(BINARY_NAME) px-service px-service.exe
