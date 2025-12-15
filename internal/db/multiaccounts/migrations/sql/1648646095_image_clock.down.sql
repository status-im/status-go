/* SQLite does not support dropping columns, hence we must create a temp table */
CREATE TEMPORARY TABLE identity_images_backup(
    key_uid VARCHAR,
    name VARCHAR,
    image_payload BLOB NOT NULL,
    width int,
    height int,
    file_size int,
    resize_target int,
    PRIMARY KEY (key_uid, name) ON CONFLICT REPLACE
) WITHOUT ROWID;

INSERT INTO identity_images_backup SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images;

DROP TABLE identity_images;


CREATE TABLE IF NOT EXISTS identity_images(
    key_uid VARCHAR,
    name VARCHAR,
    image_payload BLOB NOT NULL,
    width int,
    height int,
    file_size int,
    resize_target int,
    PRIMARY KEY (key_uid, name) ON CONFLICT REPLACE
) WITHOUT ROWID;


INSERT INTO identity_images SELECT key_uid, name, image_payload, width, height, file_size, resize_target FROM identity_images_backup;

DROP TABLE identity_images_backup;
