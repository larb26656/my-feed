# Keep Source Type definitions and Obtain in one module

The `sourcetype` module replaces `obtain` and keeps each fetchable Source Type's schema, UI metadata, normalization, key derivation, and Obtain implementation together. The `source` module owns the Source aggregate and persistence, stores the type code and normalized config with its key, and uses the `sourcetype` API to validate config and obtain an Entry Preview. Fetch reads Source data through the Source API and obtains Entries through the same `sourcetype` API as Preview.

Bootstrap registers the fetchable Source Types. The `source` module owns Saved links as a system singleton of type `manual`, which requires no registry entry or Obtain implementation.
