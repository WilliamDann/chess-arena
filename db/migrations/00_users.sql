-- stores private user information
create table users (
  id        uuid primary key default gen_random_uuid(),
  email     text unique not null,
  pw_hash   text not null
);

-- stores public user information
create table profiles (
  id      uuid primary key not null references users(id),
  display_name text
);

-- stores auth sessions for users
create table sessions (
  id         uuid primary key default gen_random_uuid(),
  user_id    uuid not null references users(id),
  created_at timestamptz not null default now(),
  expires_at timestamptz not null default now() + interval '7 days'
);
