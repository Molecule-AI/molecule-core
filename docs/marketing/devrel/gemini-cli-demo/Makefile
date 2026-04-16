.PHONY: run deps check-env

## Install Python dependency
deps:
	pip install httpx

## Verify required env vars are set before running
check-env:
	@test -n "$(PLATFORM_URL)"   || (echo "Error: PLATFORM_URL is not set"   && exit 1)
	@test -n "$(PLATFORM_TOKEN)" || (echo "Error: PLATFORM_TOKEN is not set" && exit 1)
	@test -n "$(GEMINI_API_KEY)" || (echo "Error: GEMINI_API_KEY is not set" && exit 1)

## Run the demo end-to-end
run: deps check-env
	python demo.py
