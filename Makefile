.PHONY: help frontend FORCE

help:
	@echo "make frontend - run React dev server"
	@echo "Backend commands (run from root with make backend:<target>):"
	@$(MAKE) --no-print-directory -C backend help

backend\:%: FORCE
	$(MAKE) -C backend $*

FORCE:

frontend:
	cd frontend && pnpm run dev
