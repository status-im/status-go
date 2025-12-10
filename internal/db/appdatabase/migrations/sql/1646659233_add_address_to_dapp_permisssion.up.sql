CREATE TABLE IF NOT EXISTS dapps_2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    name TEXT,
    address TEXT NOT NULL default "",
    CONSTRAINT unique_dapps_name_address UNIQUE (name,address)
);

CREATE TABLE IF NOT EXISTS permissions_2 (
    dapp_id int NOT NULL,
    permission TEXT NOT NULL,
    FOREIGN KEY(dapp_id) REFERENCES dapps_2(id) ON DELETE CASCADE
);

INSERT INTO dapps_2 SELECT NULL, name, "" FROM dapps;

INSERT INTO permissions_2 select dapps_2.id, permissions.permission from permissions join dapps_2 on dapps_2.name = permissions.dapp_name;

DROP TABLE permissions;
DROP TABLE dapps;

ALTER TABLE permissions_2 RENAME TO permissions;
ALTER TABLE dapps_2 RENAME TO dapps;