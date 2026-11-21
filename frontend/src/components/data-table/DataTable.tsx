import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type OnChangeFn,
	type PaginationState,
	type SortingState,
	useReactTable,
} from "@tanstack/react-table";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
	DataTablePagination,
	type DataTablePaginationConfig,
} from "./DataTablePagination";

export type {
	CursorPagination,
	DataTablePaginationConfig as DataTablePagination,
	PagePagination,
} from "./DataTablePagination";

type DataTableProps<TData> = {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
	getRowId?: (row: TData, index: number) => string;
	emptyMessage?: string;
	ariaLabel?: string;
	pagination?: DataTablePaginationConfig;
	sorting?: { value: SortingState; onChange: OnChangeFn<SortingState> };
};

/**
 * Headless TanStack Table rendering with optional server-driven page or cursor
 * navigation. Data fetching and cursor history remain owned by the caller.
 */
export function DataTable<TData>({
	columns,
	data,
	getRowId,
	emptyMessage = "No results",
	ariaLabel = "Data table",
	pagination,
	sorting,
}: DataTableProps<TData>) {
	const table = useReactTable({
		data,
		columns,
		getCoreRowModel: getCoreRowModel(),
		getRowId,
		manualPagination: Boolean(pagination),
		manualSorting: Boolean(sorting),
		enableMultiSort: false,
		enableSortingRemoval: false,
		onSortingChange: sorting?.onChange,
		pageCount: pagination?.mode === "page" ? pagination.pageCount : -1,
		state: {
			...(sorting ? { sorting: sorting.value } : {}),
			...(pagination
				? {
						pagination: {
							pageIndex: pagination.pageIndex,
							pageSize: pagination.pageSize,
						},
					}
				: {}),
		},
		onPaginationChange: pagination
			? (updater) => {
					const current: PaginationState = {
						pageIndex: pagination.pageIndex,
						pageSize: pagination.pageSize,
					};
					const next =
						typeof updater === "function" ? updater(current) : updater;
					if (next.pageSize !== current.pageSize) {
						pagination.onPageSizeChange(next.pageSize);
					}
					if (
						pagination.mode === "page" &&
						next.pageIndex !== current.pageIndex
					) {
						pagination.onPageChange(next.pageIndex);
					}
				}
			: undefined,
	});

	return (
		<div className="space-y-3">
			<div className="overflow-x-auto rounded-xl border">
				<table className="w-full text-left text-sm" aria-label={ariaLabel}>
					<thead className="bg-muted/60 text-muted-foreground">
						{table.getHeaderGroups().map((headerGroup) => (
							<tr key={headerGroup.id} className="border-b">
								{headerGroup.headers.map((header) => (
									<th
										key={header.id}
										colSpan={header.colSpan}
										scope="col"
										aria-sort={
											header.column.getIsSorted() === "asc"
												? "ascending"
												: header.column.getIsSorted() === "desc"
													? "descending"
													: undefined
										}
										className="px-4 py-3 font-medium"
									>
										{header.isPlaceholder ? null : sorting &&
											header.column.getCanSort() ? (
											<Button
												variant="ghost"
												size="sm"
												onClick={header.column.getToggleSortingHandler()}
											>
												{flexRender(
													header.column.columnDef.header,
													header.getContext(),
												)}
												{header.column.getIsSorted() === "asc" ? (
													<ArrowUp data-icon="inline-end" />
												) : header.column.getIsSorted() === "desc" ? (
													<ArrowDown data-icon="inline-end" />
												) : (
													<ArrowUpDown data-icon="inline-end" />
												)}
											</Button>
										) : (
											flexRender(
												header.column.columnDef.header,
												header.getContext(),
											)
										)}
									</th>
								))}
							</tr>
						))}
					</thead>
					<tbody className="divide-y">
						{table.getRowModel().rows.length ? (
							table.getRowModel().rows.map((row) => (
								<tr key={row.id} className="hover:bg-muted/30">
									{row.getVisibleCells().map((cell) => (
										<td key={cell.id} className="px-4 py-3 align-middle">
											{flexRender(
												cell.column.columnDef.cell,
												cell.getContext(),
											)}
										</td>
									))}
								</tr>
							))
						) : (
							<tr>
								<td
									colSpan={Math.max(table.getVisibleLeafColumns().length, 1)}
									className="px-4 py-10 text-center text-muted-foreground"
								>
									{emptyMessage}
								</td>
							</tr>
						)}
					</tbody>
				</table>
			</div>

			{pagination && (
				<DataTablePagination
					pagination={pagination}
					canPreviousPage={table.getCanPreviousPage()}
					canNextPage={table.getCanNextPage()}
					onFirstPage={() => table.firstPage()}
					onPreviousPage={() => table.previousPage()}
					onNextPage={() => table.nextPage()}
					onLastPage={() => table.lastPage()}
				/>
			)}
		</div>
	);
}
