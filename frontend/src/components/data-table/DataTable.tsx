import {
	type ColumnDef,
	flexRender,
	getCoreRowModel,
	type PaginationState,
	useReactTable,
} from "@tanstack/react-table";
import { Button } from "#/components/ui/button";

type PaginationBase = {
	pageIndex: number;
	pageSize: number;
	onPageSizeChange: (pageSize: number) => void;
	pageSizeOptions?: number[];
	isLoading?: boolean;
};

export type PagePagination = PaginationBase & {
	mode: "page";
	pageCount: number;
	onPageChange: (pageIndex: number) => void;
};

export type CursorPagination = PaginationBase & {
	mode: "cursor";
	hasPreviousPage: boolean;
	hasNextPage: boolean;
	onPreviousPage: () => void;
	onNextPage: () => void;
};

export type DataTablePagination = PagePagination | CursorPagination;

type DataTableProps<TData> = {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
	getRowId?: (row: TData, index: number) => string;
	emptyMessage?: string;
	ariaLabel?: string;
	pagination?: DataTablePagination;
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
}: DataTableProps<TData>) {
	const table = useReactTable({
		data,
		columns,
		getCoreRowModel: getCoreRowModel(),
		getRowId,
		manualPagination: Boolean(pagination),
		pageCount: pagination?.mode === "page" ? pagination.pageCount : -1,
		state: pagination
			? {
					pagination: {
						pageIndex: pagination.pageIndex,
						pageSize: pagination.pageSize,
					},
				}
			: undefined,
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
										className="px-4 py-3 font-medium"
									>
										{header.isPlaceholder
											? null
											: flexRender(
													header.column.columnDef.header,
													header.getContext(),
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
				<div className="flex flex-wrap items-center justify-between gap-3">
					<label className="flex items-center gap-2 text-sm text-muted-foreground">
						<span>Rows per page</span>
						<select
							className="h-9 rounded-lg border bg-background px-2 text-foreground"
							value={pagination.pageSize}
							disabled={pagination.isLoading}
							onChange={(event) =>
								pagination.onPageSizeChange(Number(event.target.value))
							}
						>
							{(pagination.pageSizeOptions ?? [10, 20, 50]).map((size) => (
								<option key={size} value={size}>
									{size}
								</option>
							))}
						</select>
					</label>
					<div className="flex items-center gap-2">
						<span className="min-w-24 text-center text-sm text-muted-foreground">
							Page {pagination.pageIndex + 1}
							{pagination.mode === "page" &&
								` of ${Math.max(pagination.pageCount, 1)}`}
						</span>
						{pagination.mode === "page" ? (
							<>
								<Button
									variant="outline"
									size="sm"
									disabled={pagination.pageIndex === 0 || pagination.isLoading}
									onClick={() => table.firstPage()}
								>
									First
								</Button>
								<Button
									variant="outline"
									size="sm"
									disabled={!table.getCanPreviousPage() || pagination.isLoading}
									onClick={() => table.previousPage()}
								>
									Previous
								</Button>
								<Button
									variant="outline"
									size="sm"
									disabled={!table.getCanNextPage() || pagination.isLoading}
									onClick={() => table.nextPage()}
								>
									Next
								</Button>
								<Button
									variant="outline"
									size="sm"
									disabled={
										pagination.pageIndex >= pagination.pageCount - 1 ||
										pagination.isLoading
									}
									onClick={() => table.lastPage()}
								>
									Last
								</Button>
							</>
						) : (
							<>
								<Button
									variant="outline"
									size="sm"
									disabled={!pagination.hasPreviousPage || pagination.isLoading}
									onClick={pagination.onPreviousPage}
								>
									Previous
								</Button>
								<Button
									variant="outline"
									size="sm"
									disabled={!pagination.hasNextPage || pagination.isLoading}
									onClick={pagination.onNextPage}
								>
									Next
								</Button>
							</>
						)}
					</div>
				</div>
			)}
		</div>
	);
}
