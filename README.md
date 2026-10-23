# portfolio-api

API for my portfolio. Serves the content as JSON, and lets me edit it and upload images after signing in with `ADMIN_PASSWORD`.

Content is stored in SQLite. A new database is seeded from `internal/store/seed.json`. The seed's screenshots live in `internal/store/seed-uploads/` and are copied into `uploads/` on every start, without overwriting files already there. To reseed a local database, run `make reseed`. To reseed in production, see `DEPLOY.md`.

Visitors can star projects and comment on them. Comments are limited to 5 per IP address every 10 minutes, and the signed-in owner can delete them.

## Run locally

```sh
cp .env.example .env
make dev     # http://localhost:8001
make test
make reseed  # delete data/portfolio.db; the next `make dev` seeds a fresh one
```

| Variable          | Notes                                                                  |
| ----------------- | ---------------------------------------------------------------------- |
| `ADMIN_PASSWORD`  | Editing is disabled when unset.                                        |
| `ALLOWED_ORIGINS` | Comma-separated origins the pages are served from (CORS).              |
| `API_ADDRESS`     | Hostname and port Caddy serves on. Only used by `compose.yaml`.        |
| `ADDR`            | Listen address. Default `:8001`.                                       |
| `DATA_DIR`        | Holds `portfolio.db` and `uploads/`. Default `./data`.                 |

## Deploy

Pushes to `main` run the tests and publish `ghcr.io/crzidev/portfolio-api:latest`. A `v1.2.3` tag also publishes `1.2.3` and `1.2`.

On the VPS, with `compose.yaml`, `Caddyfile` and `.env` in place:

```sh
docker compose pull
docker compose up -d
```

Set `API_TAG` to pin a version. Caddy serves HTTPS on 8443 (443 is taken by another app) and needs port 80 for its certificate. Data lives in the `portfolio-data` volume.

To try the compose setup locally, set `API_ADDRESS=http://localhost:8443` and run `docker compose up --build`.
