# ==== Variables ====
SERVICES := auth user
SHARED := gen pkg

# ==== Helpers ====
.PHONY: all tidy build run test

# Run tidy on all modules (services + shared)
tidy:
	@for d in $(SERVICES); do \
		echo "===> Tidying service $$d"; \
		(cd services/$$d && go mod tidy); \
	done
	@for d in $(SHARED); do \
		echo "===> Tidying shared module $$d"; \
		(cd $$d && go mod tidy); \
	done

# Build all services
build:
	@mkdir -p bin
	@for d in $(SERVICES); do \
		echo "===> Building service $$d"; \
		cd services/$$d && go build -o ../../bin/$$d ./...; \
	done

# Run a specific service: make run SERVICE=auth
run:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make run SERVICE=auth"; \
		exit 1; \
	fi
	@echo "===> Running $(SERVICE)"
	@cd services/$(SERVICE) && go run ./...

# Test all services
test:
	@for d in $(SERVICES); do \
		echo "===> Testing service $$d"; \
		cd services/$$d && go test ./...; \
	done
