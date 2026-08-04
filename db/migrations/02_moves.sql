-- stores moves for games
create table moves (
  -- move info 
  game_id    uuid not null references games(id),
  ply        int  not null,
  uci        text not null,
  created_at timestamptz not null default now(),
 
  -- sync info
  fen        text not null,
  white_ms   int not null,
  black_ms   int not null, 

  primary key (game_id, ply)
)
