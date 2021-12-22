.PHONY: build
build:
	go build -o ./awgodoc cmd/awgodoc/main.go
	
.PHONY: alfredworkflow
alfredworkflow: build
	zip awgodoc.alfredworkflow info.plist ./awgodoc
	
.PHONY: install
install: alfredworkflow
	open awgodoc.alfredworkflow