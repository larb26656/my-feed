import type { Source } from "#/features/feed/types";
import { api, type PageMeta, type SourcesPage } from "#/lib/api";

export type SourceFilters = {
	name: string;
	type: string;
};

export type SourceSortField = "name" | "type" | "createdAt";
export type SourceSortDirection = "asc" | "desc";

export type SourceSort = {
	sortBy: SourceSortField;
	sortDirection: SourceSortDirection;
};

export type SourcesPageSize = 10 | 20 | 50;

export type SourceListQuery = {
	active: boolean;
	page: number;
	pageSize: number;
	filters: SourceFilters;
	sort: SourceSort;
};

/** Query string for the shared page-based Sources API. */
export function sourcesQueryParams(query: SourceListQuery) {
	const params = new URLSearchParams({
		activeEqual: String(query.active),
		page: String(query.page),
		pageSize: String(query.pageSize),
		sortBy: query.sort.sortBy,
		sortDirection: query.sort.sortDirection,
	});
	if (query.filters.name.trim())
		params.set("nameLike", query.filters.name.trim());
	if (query.filters.type) params.set("typeEqual", query.filters.type);
	return params;
}

/** One page of Sources. Desktop pagination and mobile infinite scroll both use this. */
export async function fetchSourcesPage(
	query: SourceListQuery,
): Promise<SourcesPage> {
	const result = await api<Source[], PageMeta>(
		`/sources/?${sourcesQueryParams(query)}`,
	);
	return {
		data: result.data,
		meta: result.meta ?? {
			page: query.page,
			pageSize: query.pageSize,
			total: result.data.length,
			totalPages: 1,
		},
	};
}

export function sourcesEmptyMessage(active: boolean, filters: SourceFilters) {
	if (filters.name || filters.type) return "No sources match your filters";
	return active ? "No active sources" : "No inactive sources";
}
