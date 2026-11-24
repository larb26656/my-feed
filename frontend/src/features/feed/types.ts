export type { Entry, EntryPage, Source } from "#/lib/api";

export type EntryFilters = {
	sourceId?: string;
	savedOnly?: boolean;
	unread?: boolean;
	sortDirection: "asc" | "desc";
	sortField: "publishedAtSort" | "createdAtSort" | "savedAtSort";
};
