# Identify Bluesky Sources by DID

Readers add a Bluesky Source using a handle or profile URL, which we resolve to a DID for the Source key. A handle can change or be reassigned, so Fetch Jobs use the DID to keep following the same profile and detect duplicate Sources across handle changes. Entry links use the author's DID and post record key because the Entry's link determines its identity within a Source.

The profile association is fixed at creation so existing Entries retain their original Source. Following a different profile requires a new Source.
