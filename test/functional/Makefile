.PHONY: pytest-lint

lint:
	@echo "Running python linting on all files..."
	pre-commit run --all-files --verbose --config .pre-commit-config.yaml

compose:
	docker compose \
		-f docker-compose.anvil.yml \
		-f docker-compose.waku.yml \
		up --build --remove-orphans -d

compose-down:
	docker compose \
		-f docker-compose.anvil.yml \
		-f docker-compose.waku.yml \
		down
