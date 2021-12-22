.PHONY: build
build:
	go build -o ./awgodoc cmd/awgodoc/main.go
	
.PHONY: alfredworkflow
alfredworkflow: build
	zip godoc.alfredworkflow info.plist ./awgodoc