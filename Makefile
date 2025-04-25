BIN_NAME = datadog-sql-metics
.PHONY: lint clean test ci-lint

LINT_CMD = golangci-lint run --path-prefix .

lint:
	@echo "Running golangci-lint in the current directory..."
	@$(LINT_CMD)

# CI環境用のlintコマンド (より厳格なチェック)
ci-lint:
	@echo "Running golangci-lint for CI..."
	@golangci-lint run --timeout=5m

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_NAME)

