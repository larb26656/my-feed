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
	history: (
		status: string | undefined,
		cursor: string | null,
		pageSize: number,
	) => ["fetch-jobs", "history", status ?? "all", pageSize, cursor] as const,
	active: ["fetch-jobs", "active"] as const,
	detail: (id: string | null) => ["fetch-jobs", id] as const,
};
