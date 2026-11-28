# My feed

A personal, single-user reader that aggregates content from multiple external origins into one place.

## Language

**Entry**:
One piece of content the reader can open — typically with title and description, and optionally a link and thumbnail — that belongs to a Source. It may arrive by fetch (for example from an RSS Source) or by the reader recording a link into the manual Source. It has a source-published time (when known), a recorded time when the system accepted it, a read time when the reader marks it read, and a saved time when the reader bookmarks it. Entries with the exact same non-empty link after surrounding whitespace is removed are the same content across the reader, even if they arrive from different Sources. An Entry may have no link; Entries without a link are not considered duplicates based on their missing links. It belongs to the Source that first recorded it; another Source finding the same link does not change its metadata. Recording the same link again from its owning Source refreshes its metadata. Entries remain visible in the main stream even if their Source is deactivated. Each Entry has a stable unique id.
_Avoid_: Feed (for the readable unit), article, post, item

**Entry Preview**:
A read-only sample of content obtained from a proposed Source and shown as it would appear in the reader before the Source is created. The sample is temporary and does not become an Entry in the reader.
_Avoid_: health check, dry run (for the sample)

**Source**:
An origin of Entries, identified by a type and type-specific configuration. Some types are fetched externally (for example RSS, which requires a unique feed URL); the manual type is a single system-provided Source named "Saved links" that accepts Entries only when the reader records them. A Source may be active or deactivated: deactivated Sources are not given new Fetch Jobs and are omitted from the default Source list, but their Entries remain and still reference them. Fetchable Sources and Push Sources can be deactivated and reactivated; the manual Source cannot be deactivated. The reader creates fetchable Sources and Push Sources. Distinct from the Entries it produces. An Entry always belongs to a Source.
_Avoid_: subscription, channel, soft delete

**Source Deletion**:
The permanent removal of an active or deactivated fetchable Source or Push Source and all its Entries, including Saved Entries. The manual Source is permanent.

**Source Type**:
The kind of Source, which determines how Entries arrive and what configuration it needs (`rss`, `manual`, `youtube`, `bluesky`, `medium`, `devto`, or `push`). A `youtube` Source follows one YouTube channel, and a `bluesky` Source follows one public Bluesky profile. The `rss` type accepts both RSS and Atom documents from a URL; the document format does not define a separate Source Type. The type is fixed when the Source is created.
_Avoid_: source kind, provider, connector

**Push Source**:
A named Source created by the reader to receive Entries submitted by an external producer such as an AI workflow. It accepts submissions while active and retains its Entries when deactivated.

**Pushed Entry**:
An Entry submitted to a Push Source, such as a standalone summary with optional reference links. Submitting the same producer-assigned Entry identifier to the same Push Source updates the existing Entry and preserves its read and Saved state.

**Entry Body**:
The full content of an Entry that the reader can open inside My Feed, with an optional link to the original. The Entry description is the short summary shown in the stream.

**Source Config**:
The type-specific settings that describe where a Source gets its Entries, such as an RSS feed URL or a YouTube Channel ID. The reader can change these settings for a fetchable Source while keeping its identity, Source Type, and existing Entries.

**Bluesky Source**:
A Source that follows one public Bluesky profile across handle changes. Its Entries come from the profile's own top-level posts, including quote posts.

**Medium Source**:
A Source that follows one Medium Profile by username or one Medium Publication by its publication slug.

**DEV.to Source**:
A Source that follows one tag or one account on DEV.to. An account can be an individual or an organization and is selected by username.

**Source Profile Image**:
An optional image chosen by the reader to represent a fetchable Source, whether active or deactivated. The manual Source cannot have a Source Profile Image.
_Avoid_: thumbnail (for a Source)

**Saved**:
A reader bookmark on an Entry (the time the reader marked it saved). Distinct from recording a new Entry from a link the reader found.
_Avoid_: read later, save for later, bookmark (as a separate entity)

**Feed**:
Reserved for a future ranked or algorithmic stream of Entries (the ordered view the reader sees). Not an entity in the current model.
_Avoid_: using Feed for a single piece of content or for a Source

**Fetch Job**:
One unit of work that obtains content from a single fetchable Source, maps it into Entry fields, and records those Entries. It progresses through pending, running, succeeded, or failed; retries stay on the same Fetch Job until it succeeds or reaches its maximum attempts (a property of that Fetch Job; the product currently uses 3). Each attempt runs the whole obtain→map→record path; there is at most one unfinished Fetch Job per Source. Succeeded means that path completed, even when no Entry was newly created; the Fetch Job records how many Entries it newly created (metadata refresh of an existing Entry does not count). A terminal failed Fetch Job is finished; the next trigger creates a new Fetch Job. If the Source is deactivated while a Fetch Job is pending, that job becomes failed; if it is already running, it is allowed to finish. Deactivated Sources do not get new Fetch Jobs. Fetch Jobs are created on a fixed-rate round or manually for one Source. There is no separate persisted staging entity between origin content and Entry.
_Avoid_: Job (alone), Fetcher, Fetch Run, poll, sync, Fetch Schedule

**Session**:
One login of the single Reader on a particular client. It lasts until logout, expiry, or revocation. The same Reader may hold multiple Sessions at once (for example several browsers or devices). Calling the API with the shared API key is not a Session.
_Avoid_: User, Admin, account (as an entity), login (as the lasting thing)
