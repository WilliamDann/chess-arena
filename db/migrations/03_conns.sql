create unlogged table connections (
  id         uuid primary key default gen_random_uuid(),
  user_id    uuid not null references users(id),
  server_id  uuid not null,
  last_seen  timestamptz not null default now()
)
