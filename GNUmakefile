default: testacc

# Build the provider binary to dist/
.PHONY: build
build:
	go build -o ./dist/

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
