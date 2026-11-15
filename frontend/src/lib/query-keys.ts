import type { EntryFilters } from "#/features/feed/types";

export const sourceKeys = {
	all: ["sources"] as const,
};

export const entryKeys = {
	root: ["entries"] as const,
	list: (filters: EntryFilters) => ["entries", filters] as const,
	savedCount: ["saved-count"] as const,
};

export const fetchJobKeys = {
	root: ["fetch-jobs"] as const,
	detail: (id: string | null) => ["fetch-jobs", id] as const,
};
