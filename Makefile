ifeq ($(OS),Windows_NT)
    SHELL := pwsh.exe
    .SHELLFLAGS := -NoProfile -NonInteractive -Command

	outfile=pbmanager.exe
else
    SHELL := /usr/bin/env pwsh
    .SHELLFLAGS := -NoProfile -NonInteractive -Command

	outfile=pbmanager.exe
endif

build_time = $(shell pwsh -NoProfile -NonInteractive -Command "(Get-Date).ToString('yyyy-MM-ddTHH:mm:sszzz')")
ldflags=-X 'github.com/informaticon/dev.win.base.pbmanager/cmd.Version=0.0.0-trunk' \
		-X 'github.com/informaticon/dev.win.base.pbmanager/cmd.BuildTime=$(build_time)'

.PHONY: build
build:
	$$env:GOOS = "windows"; go build -o "$(outfile)" -ldflags "$(ldflags)"

.PHONY: test
tests:
	go test ./...

.PHONY: install
install: build
ifneq ($(BIN_INSTALL_PATH),)
	cp "$(outfile)" "$(BIN_INSTALL_PATH)"
else
	@echo "environment variable BIN_INSTALL_PATH must be set to use the install target"
endif

	

generateFileStorageInterface:
	oapi-codegen -generate types,client -o ./dist/filestorage/fileapi.go -package filestorage ./dist/filestorage/FileServiceOpenApi.yaml