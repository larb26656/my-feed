import { DataTable } from "#/components/data-table/DataTable";
import { Button } from "#/components/ui/button";
import type { SourceActionHandlers } from "#/features/sources/SourceActions";
import {
	type SourceFilters,
	type SourceSort,
	sourcesEmptyMessage,
} from "#/features/sources/source.api";
import { sourceColumns } from "./SourceColumns";
import { useSourceTable } from "./useSourceTable";

export function SourceTableView({
	active,
	filters,
	sort,
	onSortChange,
	actions,
}: {
	active: boolean;
	filters: SourceFilters;
	sort: SourceSort;
	onSortChange: (sort: SourceSort) => void;
	actions: SourceActionHandlers;
}) {
	const table = useSourceTable({ active, filters, sort, onSortChange });
	const sources = table.query.data?.data ?? [];
	const totalPages = Math.max(table.query.data?.meta.totalPages ?? 1, 1);
	const label = active ? "active" : "inactive";

	return (
		<div className="mt-4">
			<DataTable
				columns={sourceColumns(actions)}
				data={sources}
				isLoading={table.query.isFetching}
				error={
					table.query.isError
						? {
								error: table.query.error ?? "Sources unavailable",
								message: "Could not load sources. Please try again",
								action: (
									<Button onClick={() => void table.query.refetch()}>
										Try again
									</Button>
								),
							}
						: undefined
				}
				getRowId={(source) => source.id}
				ariaLabel={`${label} sources`}
				sorting={{
					value: table.sorting,
					onChange: table.onSortingChange,
				}}
				emptyMessage={sourcesEmptyMessage(active, filters)}
				pagination={{
					mode: "page",
					pageIndex: table.pageIndex,
					pageCount: totalPages,
					pageSize: table.pageSize,
					pageSizeOptions: [...table.pageSizeOptions],
					isLoading: table.query.isFetching,
					onPageChange: table.setPageIndex,
					onPageSizeChange: table.onPageSizeChange,
				}}
			/>
		</div>
	);
}
