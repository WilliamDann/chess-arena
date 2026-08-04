create type game_result as enum ('*', '1-0', '1/2-1/2', '0-1');

create table challenges (
  id                  uuid primary key default gen_random_uuid(),
  from_player         uuid not null references users(id),
  to_player           uuid          references users(id),

  clock_initial_ms    int not null,
  clock_increment_ms  int not null default 0,

  created_at          timestamptz not null default now()
);

create index challenges_from_player_idx on challenges (from_player);
create index challenges_to_player_idx   on challenges (to_player);

create table games (
  id                  uuid primary key default gen_random_uuid(),
  white_player        uuid not null references users(id),
  black_player        uuid not null references users(id),

  result              game_result not null default '*',

  clock_initial_ms    int not null,
  clock_increment_ms  int not null default 0,

  created_at          timestamptz not null default now(),
  result_at           timestamptz
);

create index games_white_player_idx on games (white_player);
create index games_black_player_idx on games (black_player);

create index games_active_white_idx on games (white_player) where result = '*';
create index games_active_black_idx on games (black_player) where result = '*';
