# Accept externally produced Entries through Push Sources

AI workflows and other external producers submit to named Push Sources through MCP or HTTP. The destination Source and the producer-assigned Entry identifier determine which Entry to create or update, so retries can preserve reader state even when an Entry has no URL or several Entries refer to the same original.

Entry Body carries full Markdown content separately from the short description and the optional original URL, so summaries and future extracted articles can be read in My Feed. An Entry with a body opens in a full-screen reader with access to the original when available. An Entry with only a URL opens the original.
