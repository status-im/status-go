CREATE TABLE registrations (
    counter INTEGER PRIMARY KEY AUTOINCREMENT,
    peer VARCHAR(64),
    ns VARCHAR, 
    expire INTEGER, 
    addrs VARBINARY
);

CREATE TABLE nonce (
    nonce VARBINARY
);
