export type { Entry, EntryPage, Source } from "#/lib/api";

export type EntryFilters = {
	sourceId?: string;
	savedOnly: boolean;
	unread: boolean;
};

export const filters = ["All", "Unread"] as const;
export type FeedFilter = (typeof filters)[number];
