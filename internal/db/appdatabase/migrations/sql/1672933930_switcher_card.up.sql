CREATE TABLE switcher_cards (
  card_id TEXT PRIMARY KEY ON CONFLICT REPLACE,
  type INT NOT NULL DEFAULT 0,
  clock INT NOT NULL,
  screen_id TEXT DEFAULT ""
);
