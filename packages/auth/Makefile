# Service name
SERVICE_NAME = auth-service

BIN_DIR = bin

BIN_PATH = ${BIN_DIR}/${SERVICE_NAME}



# Default build flags
GO_FLAGS = -ldflags "-X main.commit=$(COMMIT_SHA) -X main.migration=$(EXPECTED_MIGRATION_TIMESTAMP)"
GO_FILES = ./...

# Default ports (can be overridden at runtime)
AUTH_PORT ?= 50051
METRICS_PORT ?= 9091

# === Targets ===
.PHONY: all build run tidy test clean
all: build

## Build binary
build:
	@echo ">> Building $(SERVICE_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build $(GO_FLAGS) -o $(BIN_PATH) .

## Run service locally (requires .env + config.yaml)
run: build
	@echo ">> Running $(SERVICE_NAME) on ports $(AUTH_PORT) and $(METRICS_PORT)..."
	@./$(BIN_PATH) --config=config.yaml

## Run tests
test:
	@echo ">> Running tests..."
	@go test -v $(GO_FILES)

## Clean build artifacts
clean:
	@echo ">> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)

## Tidy modules
tidy:
	@echo ">> Tidying Go modules..."
	@go mod tidy