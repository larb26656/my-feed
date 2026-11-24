import { useInfiniteQuery } from "@tanstack/react-query";
import {
	fetchSourcesPage,
	type SourceFilters,
	type SourceSort,
} from "#/features/sources/source.api";
import { sourceKeys } from "#/lib/query-keys";

/** Page size for the phone list. The API is still the shared page endpoint. */
export const mobileSourcePageSize = 20;

export function useInfiniteSources(
	active: boolean,
	filters: SourceFilters,
	sort: SourceSort,
) {
	return useInfiniteQuery({
		queryKey: sourceKeys.infinite(active, mobileSourcePageSize, filters, sort),
		initialPageParam: 1,
		queryFn: ({ pageParam }) =>
			fetchSourcesPage({
				active,
				page: pageParam,
				pageSize: mobileSourcePageSize,
				filters,
				sort,
			}),
		getNextPageParam: (last, _pages, lastPageParam) => {
			const page = last.meta.page ?? lastPageParam;
			const totalPages = last.meta.totalPages ?? 1;
			return page < totalPages ? page + 1 : undefined;
		},
	});
}
