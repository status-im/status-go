ALTER TABLE communities_communities ADD COLUMN control_node BLOB;
CREATE TABLE communities_validate_signer (
  id BLOB NOT NULL,
  clock INT NOT NULL,
  payload BLOB NOT NULL,
  validate_at INT NOT NULL,
  signer BLOB NOT NULL,
  PRIMARY KEY(id, signer) ON CONFLICT REPLACE
);

CREATE INDEX communities_validate_signer_clock ON communities_validate_signer(validate_at, clock);
