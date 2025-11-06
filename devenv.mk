##@ devenv

DEVENV ?= devenv

.PHONY: devenv-shell devenv-update

devenv-shell: ## Activate the developer environment. https://devenv.sh/basics/
	@$(DEVENV) shell

devenv-update: ## Update the developer environment. https://devenv.sh/basics/
	@$(DEVENV) update