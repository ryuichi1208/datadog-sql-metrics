BIN_NAME = datadog-sql-metics
.PHONY: lint clean

LINT_CMD = golangci-lint run --path-prefix .

lint:
	@echo "Running golangci-lint in the current directory..."
	@$(LINT_CMD)

clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_NAME)

