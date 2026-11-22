# ADR 0006 — Delete a Source and its history

## Status

Accepted

## Decision

The Reader can permanently delete an active or deactivated RSS or YouTube Source. Deletion removes all of its Entries, including Saved Entries, while preserving the rule that each Entry belongs to a Source. Deactivation is available when the Reader wants to stop new Fetch Jobs and keep existing Entries. The manual Source is permanent.

Before the Sources UI asks the Reader to confirm, the backend counts up to 100 Entries for the Source, including Saved Entries. The query applies `LIMIT 100` before counting and uses the existing `(source_id, link)` index without ordering. The preview returns `entryCount` as an integer from 0 to 100, where 100 means at least 100 Entries. The dialog displays counts from `0` through `99` as numbers and displays `+99` for 100. A nonzero count links to the existing Source page in a new tab so the Reader can inspect the Entries and return to the dialog. The count reflects the time of the read. The server keeps no pending confirmation state, and a confirmed deletion applies to the Source and its Entries when the deletion runs.

The deletion transaction removes the Source, its Entries, and its Fetch Jobs, including pending jobs and finished history. If a Fetch Job is running, deletion returns a conflict and the Reader can retry after the job finishes. This allows the executor to finish without losing the Source or the Fetch Job it needs to complete.

After the transaction commits, the application removes the Source Profile Image file if one exists. It logs cleanup failures and reports the Source deletion as successful, following the existing profile image removal behavior.
