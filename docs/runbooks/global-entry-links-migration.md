# Global Entry link migration

Migration 000007 consolidates Entries and cannot be reversed by a code rollback. Stop the application and all other writers for the entire backup, migration, and verification window. Use a SQLite-consistent backup (for example `sqlite3 library.db '.backup pre-global-links.db'`), then run `PRAGMA integrity_check` and `PRAGMA foreign_key_check` against the backup and verify it can be opened. Keep the backup separately from the live database.

Before upgrading, inspect `sqlite_schema` for foreign keys or other external references to `entries.id`. This repository has no such foreign keys, but external consumers may retain IDs. Resolve those references before migration; discarded IDs cannot be inferred afterwards without a pre-migration copy. Record duplicate groups and affected counts using the same whitespace set as migration 000007:

```sql
WITH normalized AS (SELECT id, NULLIF(trim(link, char(9,10,11,12,13,32,133,160,5760,8192,8193,8194,8195,8196,8197,8198,8199,8200,8201,8202,8232,8233,8239,8287,12288)), '') AS link FROM entries)
SELECT link, count(*) AS entries FROM normalized WHERE link IS NOT NULL GROUP BY link HAVING count(*) > 1;
```

Retain the backup to compare each group's winner: Saved first, then lowest creation time and ID; the winner retains its Source and metadata, with the minimum non-NULL read and Saved timestamps. Start the upgraded application to run the transactional migration. Before reopening writes, check `PRAGMA integrity_check`, `PRAGMA foreign_key_check`, and `SELECT link, count(*) FROM entries WHERE link IS NOT NULL GROUP BY link HAVING count(*) > 1` (no rows). Compare winners and timestamps with the backup; verify multiple NULL links are permitted on a disposable copy.

If migration or verification fails, keep writers stopped and restore the verified backup using SQLite's backup/restore procedure, including any WAL files as appropriate. If writes have occurred since backup, preserve the current database separately and reconcile those writes against the restored copy before reopening service. A rollback of application code or a Goose down migration cannot restore discarded Entries.
