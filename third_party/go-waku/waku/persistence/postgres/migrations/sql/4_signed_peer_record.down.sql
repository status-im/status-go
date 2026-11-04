DROP TABLE registrations;

CREATE TABLE registrations (
    counter SERIAL PRIMARY KEY,
    peer VARCHAR(64),
    ns VARCHAR, 
    expire INTEGER, 
    addrs BYTEA
);