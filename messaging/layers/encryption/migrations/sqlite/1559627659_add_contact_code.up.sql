CREATE TABLE contact_code_config (
   unique_constraint varchar(1) NOT NULL PRIMARY KEY DEFAULT 'X',
  last_published INTEGER NOT NULL DEFAULT 0
);

INSERT INTO contact_code_config VALUES ('X', 0);
