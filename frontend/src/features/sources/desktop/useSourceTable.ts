import { useQuery } from "@tanstack/react-query";
import type { OnChangeFn, SortingState } from "@tanstack/react-table";
import { useState } from "react";
import {
	fetchSourcesPage,
	type SourceFilters,
	type SourceSort,
	type SourceSortField,
	type SourcesPageSize,
} from "#/features/sources/source.api";
import { sourceKeys } from "#/lib/query-keys";

const pageSizeOptions = [
	10, 20, 50,
] as const satisfies readonly SourcesPageSize[];

export { pageSizeOptions };

function toSortingState(sort: SourceSort): SortingState {
	return [{ id: sort.sortBy, desc: sort.sortDirection === "desc" }];
}

function fromSortingState(state: SortingState): SourceSort {
	const id = state[0]?.id;
	const sortBy: SourceSortField =
		id === "type" || id === "createdAt" || id === "name" ? id : "name";
	return {
		sortBy,
		sortDirection: state[0]?.desc ? "desc" : "asc",
	};
}

export function useSourceTable({
	active,
	filters,
	sort,
	onSortChange,
}: {
	active: boolean;
	filters: SourceFilters;
	sort: SourceSort;
	onSortChange: (sort: SourceSort) => void;
}) {
	const [pageIndex, setPageIndex] = useState(0);
	const [pageSize, setPageSize] = useState<SourcesPageSize>(10);
	const criteria = `${active}|${filters.name}|${filters.type}|${sort.sortBy}|${sort.sortDirection}`;
	const [pageCriteria, setPageCriteria] = useState(criteria);
	// Drop back to the first page before the query runs when the shared criteria change.
	const page = pageCriteria === criteria ? pageIndex : 0;
	if (pageCriteria !== criteria) {
		setPageCriteria(criteria);
		setPageIndex(0);
	}

	const query = useQuery({
		queryKey: sourceKeys.page(active, page + 1, pageSize, filters, sort),
		queryFn: () =>
			fetchSourcesPage({
				active,
				page: page + 1,
				pageSize,
				filters,
				sort,
			}),
		placeholderData: (previous) => previous,
	});

	const onSortingChange: OnChangeFn<SortingState> = (updater) => {
		const current = toSortingState(sort);
		const next = typeof updater === "function" ? updater(current) : updater;
		onSortChange(fromSortingState(next));
	};

	return {
		query,
		pageIndex: page,
		pageSize,
		pageSizeOptions,
		setPageIndex,
		onPageSizeChange: (next: number) => {
			setPageSize(next as SourcesPageSize);
			setPageIndex(0);
		},
		sorting: toSortingState(sort),
		onSortingChange,
	};
}
