export type { Entry, EntryPage, Source } from "#/lib/api";

export type EntryFilters = {
	sourceId?: string;
	savedOnly: boolean;
	unread: boolean;
	sortDirection: "asc" | "desc";
};

export const filters = ["All", "Unread"] as const;
export type FeedFilter = (typeof filters)[number];
