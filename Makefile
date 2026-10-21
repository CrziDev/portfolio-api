# Loads .env, then runs the API.
dev:
	set -a; . ./.env; set +a; go run .

test:
	go test ./...

.PHONY: dev test
