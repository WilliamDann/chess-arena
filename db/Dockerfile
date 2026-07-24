FROM postgres:17-alpine

# Files in /docker-entrypoint-initdb.d run alphabetically on first startup,
# so migrations run before the zz-prefixed seed script.
COPY migrations/ /docker-entrypoint-initdb.d/
COPY scripts/seed.sql /docker-entrypoint-initdb.d/zz-seed.sql
