# generate-db

A tiny utility to generate empty, up-to-date SQLite databases used by Status. 

It initializes the databases using the same initializers and migrations as the app, 
so the resulting files match the current schema. This is handy for IDE SQL inspections, 
schema exploration, or tooling that expects a real database file.

## What it creates

- `app.db` — initialized via `appdatabase.DbInitializer`
- `wallet.db` — initialized via `walletdatabase.DbInitializer`
- `accounts.db` — initialized via `multiaccounts.InitializeDB`

By design, these databases are empty (no user data) but fully migrated to the latest schema.

## Build and run

```bash
make generate-db
```
This builds the tool and generates databases under `build/db` by default.

## Notes

- Schemas are produced by the current codebase.  
  If you update migrations, rerun this tool to regenerate the files.
- The tool uses an empty password and 0 KDF iterations during initialization.  
  These files are strictly for local development, inspection, and tooling.
- Existing files at the target location are removed before re-creation. 
