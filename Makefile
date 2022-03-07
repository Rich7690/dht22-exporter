build:
	GOARCH=arm GOARM=5 go build -o main.armv5 && GOARCH=arm go build -o main.arm
download:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.42.0
lint:
	./bin/golangci-lint run --fix
.PHONY: build
build: ## goreleaser --snapshot --skip-publish --rm-dist
build: 
	goreleaser --snapshot --skip-publish --rm-dist