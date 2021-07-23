BUILD_TAGS := -tags netgo
BUILD_FLAGS := -ldflags '-extldflags "-static"' -trimpath

.PHONY: build
build:
	go mod tidy
	$(GO_MODULE) go build $(BUILD_TAGS) $(BUILD_FLAGS) -o . ./cmd/...

