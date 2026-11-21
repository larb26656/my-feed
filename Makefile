.PHONY: help frontend hooks FORCE

help:
	@echo "make frontend - run React dev server"
	@echo "make hooks    - install Git hooks (requires Lefthook)"
	@echo "Backend commands (run from root with make backend:<target>):"
	@$(MAKE) --no-print-directory -C backend help

backend\:%: FORCE
	$(MAKE) -C backend $*

FORCE:

frontend:
	cd frontend && pnpm run dev

hooks:
	lefthook install
