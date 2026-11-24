.PHONY: help frontend storybook build-storybook hooks generate-api-client check-api-client FORCE

help:
	@echo "make frontend - run React dev server"
	@echo "make storybook - run Storybook dev server"
	@echo "make build-storybook - build Storybook"
	@echo "make hooks    - install Git hooks (requires Lefthook)"
	@echo "make generate-api-client - regenerate FE OpenAPI types from swagger.json"
	@echo "make check-api-client    - fail if FE OpenAPI types drift from swagger.json"
	@echo "Backend commands (run from root with make backend:<target>):"
	@$(MAKE) --no-print-directory -C backend help

backend\:%: FORCE
	$(MAKE) -C backend $*

FORCE:

frontend:
	cd frontend && pnpm run dev

storybook:
	cd frontend && pnpm run storybook

build-storybook:
	cd frontend && pnpm run build-storybook

generate-api-client:
	cd frontend && pnpm openapi:generate

check-api-client:
	cd frontend && pnpm openapi:check

hooks:
	lefthook install
