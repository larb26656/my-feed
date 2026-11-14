# Quick Feed

A personal, single-user reader that aggregates content from multiple external origins into one place.

## Language

**Entry**:
One piece of content the reader can open — typically with title, description, link, and optional thumbnail — ingested from a Source. It has a source-published time (when known), a recorded time when the system accepted it, a read time when the reader marks it read, and a saved time when the reader bookmarks it. Within a Source, an Entry is the same content when it shares the same link. Each Entry has a stable unique id.
_Avoid_: Feed (for the readable unit), article, post, item

**Source**:
An external origin the system fetches from (for example an RSS feed URL or a scrape target). Distinct from the Entries it produces. An Entry always belongs to a Source.
_Avoid_: subscription, channel

**Feed**:
Reserved for a future ranked or algorithmic stream of Entries (the ordered view the reader sees). Not an entity in the current model.
_Avoid_: using Feed for a single piece of content or for a Source
