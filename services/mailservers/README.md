Mailservers Service
================

Mailservers service provides read/write API for `Mailserver` object 
which stores details about user's mailservers.

To enable this service, include `mailservers` in APIModules:


```json
{
  "MailserversConfig": {
    "Enabled": true
  },
  "APIModules": "mailservers"
}
```

API
---

Enabling service will expose three additional methods:

#### mailservers_addMailserver

Stores `Mailserver` in the database.
All fields are specified below:

```json
{
  "id": "1",
  "name": "my mailserver",
  "address": "enode://...",
  "password": "some-pass",
  "fleet": "beta"
}
```

#### mailservers_getMailservers

Reads all saved mailservers.

#### mailservers_deleteMailserver

Deletes a mailserver specified by an ID.
