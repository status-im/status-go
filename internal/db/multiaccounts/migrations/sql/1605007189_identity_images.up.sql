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
