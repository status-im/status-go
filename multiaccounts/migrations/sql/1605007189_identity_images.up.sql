CREATE TABLE IF NOT EXISTS identity_images(
    keyUid VARCHAR,
    name VARCHAR,
    image_payload BLOB NOT NULL,
    width int,
    height int,
    file_size int,
    resize_target int,
    PRIMARY KEY (keyUid, name) ON CONFLICT REPLACE
) WITHOUT ROWID;
