
.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Show available commands
	@awk 'BEGIN { \
	    FS = ":.*##"; \
	    printf "\n\033[1;34m============================================\033[0m\n"; \
	    printf "\033[1;34m     GolangCI Lint Utility Commands\033[0m\n"; \
	    printf "\033[1;34m============================================\033[0m\n\n"; \
	    printf "\033[1mUsage:\033[0m  make \033[36m<target>\033[0m  [N=<num>]\n"; \
	} \
	/^##@/ { printf "\n\033[1;33m%s\033[0m\n", substr($$0, 5) } \
	/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 }' \
	$(MAKEFILE_LIST)
	@printf "\n\033[1;34m--------------------------------------------\033[0m\n"
	@printf "\033[1mExamples:\033[0m\n\n"
	@printf "  make lint\n"
	@printf "  make lint-fixed-plan N=7\n"
	@printf "  make lint-fixed-plan-result N=2\n"
	@printf "  make shellcheck\n"
	@printf "\n\033[1;34m============================================\033[0m\n\n"

##@ Lint

.PHONY: lint
lint: ## Run golangci-lint and generate all reports (JSON, MD, HTML, SARIF)
	@cd .. && bash golangci/cmd/golangci-report.sh

##@ Fix Planning

.PHONY: lint-fixed-plan
lint-fixed-plan: ## Generate a Markdown fix plan for issue #N  [N=<num>]
	@if [ -z "$(N)" ]; then \
		echo "Error: issue number required."; \
		echo "Usage: make lint-fixed-plan N=<number>"; \
		echo "Example: make lint-fixed-plan N=12"; \
		exit 1; \
	fi
	@cd .. && bash golangci/cmd/lint-fixed-plan.sh "$(N)"

.PHONY: lint-fixed-plan-result
lint-fixed-plan-result: ## Display and verify the fix result for issue #N  [N=<num>]
	@if [ -z "$(N)" ]; then \
		echo "Error: issue number required."; \
		echo "Usage: make lint-fixed-plan-result N=<number>"; \
		echo "Example: make lint-fixed-plan-result N=12"; \
		exit 1; \
	fi
	@cd .. && bash golangci/cmd/lint-fixed-plan-result.sh "$(N)"

##@ Quality

.PHONY: shellcheck
shellcheck: ## Run shellcheck on all shell scripts in golangci/cmd/
	@if command -v shellcheck > /dev/null 2>&1; then \
		cd .. && shellcheck golangci/cmd/golangci-report.sh \
		           golangci/cmd/lint-fixed-plan.sh \
		           golangci/cmd/lint-fixed-plan-result.sh \
		  && echo "shellcheck passed"; \
	else \
		echo "shellcheck not installed — see https://github.com/koalaman/shellcheck"; \
		exit 1; \
	fi
