# ADR 0004 — Provider-owned module API contracts

Source and Entry own the in-process contracts Fetch uses to read Source data and ingest Entries. Each contract exposes only the operations and data its callers need, which lets the owning module evolve its persistence and application logic behind that boundary. Bootstrap wires Fetch to those interfaces. An existing service can satisfy the contract directly when its signatures and behavior match, and the owning module can extend its service or provide an adapter when needed.

This decision covers module-to-module calls. See the [cross-module use case pattern](../patterns/application.md#cross-module-use-case) for transaction orchestration.
