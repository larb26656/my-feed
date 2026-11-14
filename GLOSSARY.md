# Quick Feed

A personal, single-user reader that aggregates content from multiple external origins into one place.

## Language

**Entry**:
One piece of content the reader can open — typically with title, description, link, and optional thumbnail — that belongs to a Source. It may arrive by fetch (for example from an RSS Source) or by the reader recording a link into the manual Source. It has a source-published time (when known), a recorded time when the system accepted it, a read time when the reader marks it read, and a saved time when the reader bookmarks it. Within a Source, an Entry is the same content when it shares the same link; recording the same link again refreshes that Entry's metadata. Entries remain visible in the main stream even if their Source is deactivated. Each Entry has a stable unique id.
_Avoid_: Feed (for the readable unit), article, post, item

**Source**:
An origin of Entries, identified by a type and type-specific configuration. Some types are fetched externally (for example RSS, which requires a unique feed URL); the manual type is a single system-provided Source named "Saved links" that accepts Entries only when the reader records them. A Source may be active or deactivated: deactivated Sources are not fetched and are omitted from the default Source list, but their Entries remain and still reference them. Fetchable Sources can be deactivated and reactivated; the manual Source cannot be deactivated. The reader creates only fetchable Sources. Distinct from the Entries it produces. An Entry always belongs to a Source.
_Avoid_: subscription, channel, soft delete

**Source Type**:
The kind of Source, which determines how Entries arrive and what configuration it needs (for example `rss` or `manual`). The type is fixed when the Source is created.
_Avoid_: source kind, provider, connector

**Saved**:
A reader bookmark on an Entry (the time the reader marked it saved). Distinct from recording a new Entry from a link the reader found.
_Avoid_: read later, save for later, bookmark (as a separate entity)

**Feed**:
Reserved for a future ranked or algorithmic stream of Entries (the ordered view the reader sees). Not an entity in the current model.
_Avoid_: using Feed for a single piece of content or for a Source
