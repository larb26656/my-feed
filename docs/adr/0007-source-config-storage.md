# Store Source Config as JSON with a per-type key

Store Source Config as JSON so Source Types can introduce their own fields without adding columns to the Source table. Store a separate `key` alongside the config and enforce uniqueness on `(type, key)` so each Source Type can define which config values identify the same origin while SQLite enforces uniqueness across concurrent writes.

Derive the key from the normalized config and write both atomically whenever config changes. RSS uses the trimmed feed URL, YouTube uses the canonical Channel ID, and the manual Source uses a fixed singleton key. Deactivated Sources retain their keys. Backfill existing config and keys from stored values so historical data remains readable under newer validation rules.

Editing a Source keeps its ID and existing Entries. Each Fetch Job attempt uses the config it loads at the start of the attempt, so an edit takes effect on subsequent attempts and work already underway can finish with the config it loaded.
