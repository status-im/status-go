CREATE TABLE IF NOT EXISTS social_links_settings (
  link_text TEXT PRIMARY KEY ON CONFLICT REPLACE,
  link_url TEXT
);

INSERT INTO social_links_settings (
  link_text,
  link_url
)
VALUES
  ("__twitter", NULL),
  ("__personal_site", NULL),
  ("__github", NULL),
  ("__youtube", NULL),
  ("__discord", NULL),
  ("__telegram", NULL);
