APP_NAME    := devagent
VERSION     := 0.1.0
GO          := go
BUILD_DIR   := bin
LDFLAGS     := -s -w -X 'main.version=$(VERSION)'
GOFLAGS     := -trimpath

.PHONY: all build clean test lint run help

all: build

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) .

clean:
	rm -rf $(BUILD_DIR)

test:
	$(GO) test ./... -v

lint:
	golangci-lint run ./...

run: build
	$(BUILD_DIR)/$(APP_NAME) $(ARGS)

help:
	@echo "Usage:"
	@echo "  make build        Build the binary to $(BUILD_DIR)/$(APP_NAME)"
	@echo "  make clean        Remove build artifacts"
	@echo "  make test         Run all tests"
	@echo "  make lint         Run golangci-lint"
	@echo "  make run          Build and run (pass flags via ARGS=...)"
	@echo "  make help         Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make run ARGS='-project ./myapp -task \"add tests\"'"
	@echo "  make build VERSION=1.0.0"
