# Loads .env, then runs the API.
dev:
	set -a; . ./.env; set +a; go run .

test:
	go test ./...

# Deletes the local database, so the next `make dev` seeds a fresh one from internal/store/seed.json.
# Uploaded images stay; the seed's own images are copied back into data/uploads on start.
reseed:
	rm -f data/portfolio.db data/portfolio.db-wal data/portfolio.db-shm

.PHONY: dev test reseed
