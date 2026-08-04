-- stores moves for games
create table moves (
  game_id    uuid not null references games(id),
  ply        int  not null,
  ci        text not null,

  created_at timestamptz not null default now(),

  primary key (game_id, ply)
)
