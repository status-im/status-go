CREATE TABLE IF NOT EXISTS identity_images(
    type VARCHAR,
    image_payload BLOB NOT NULL,
    width int,
    height int
    filesize int,
    resize_target int,
) WITHOUT ROWID;

