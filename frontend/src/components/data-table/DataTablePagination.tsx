import {
	ChevronLeft,
	ChevronRight,
	ChevronsLeft,
	ChevronsRight,
} from "lucide-react";
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

export type DataTablePaginationConfig = PagePagination | CursorPagination;

type DataTablePaginationProps = {
	pagination: DataTablePaginationConfig;
	canPreviousPage: boolean;
	canNextPage: boolean;
	onFirstPage: () => void;
	onPreviousPage: () => void;
	onNextPage: () => void;
	onLastPage: () => void;
};

type PageItem =
	| { kind: "page"; page: number }
	| { kind: "ellipsis"; id: "left" | "right" };

/**
 * Builds a compact page list: always first/last, current ± siblingCount,
 * with ellipsis where the range jumps.
 */
function getPageItems(
	pageCount: number,
	pageIndex: number,
	siblingCount = 1,
): PageItem[] {
	const total = Math.max(pageCount, 1);
	const current = Math.min(Math.max(pageIndex, 0), total - 1);
	const windowSize = siblingCount * 2 + 5; // first, last, current, 2 siblings, 2 ellipsis slots

	if (total <= windowSize) {
		return Array.from({ length: total }, (_, page) => ({
			kind: "page" as const,
			page,
		}));
	}

	const leftSibling = Math.max(current - siblingCount, 1);
	const rightSibling = Math.min(current + siblingCount, total - 2);
	const showLeftEllipsis = leftSibling > 1;
	const showRightEllipsis = rightSibling < total - 2;

	const items: PageItem[] = [{ kind: "page", page: 0 }];

	if (showLeftEllipsis) {
		items.push({ kind: "ellipsis", id: "left" });
	} else {
		for (let page = 1; page < leftSibling; page += 1) {
			items.push({ kind: "page", page });
		}
	}

	for (let page = leftSibling; page <= rightSibling; page += 1) {
		items.push({ kind: "page", page });
	}

	if (showRightEllipsis) {
		items.push({ kind: "ellipsis", id: "right" });
	} else {
		for (let page = rightSibling + 1; page < total - 1; page += 1) {
			items.push({ kind: "page", page });
		}
	}

	items.push({ kind: "page", page: total - 1 });
	return items;
}

export function DataTablePagination({
	pagination,
	canPreviousPage,
	canNextPage,
	onFirstPage,
	onPreviousPage,
	onNextPage,
	onLastPage,
}: DataTablePaginationProps) {
	const isLoading = Boolean(pagination.isLoading);
	const pageItems =
		pagination.mode === "page"
			? getPageItems(pagination.pageCount, pagination.pageIndex)
			: [];

	return (
		<div className="flex flex-wrap items-center justify-between gap-3">
			<label className="flex items-center gap-2 text-sm text-muted-foreground">
				<span>Rows per page</span>
				<select
					className="h-9 rounded-lg border bg-background px-2 text-foreground"
					value={pagination.pageSize}
					disabled={isLoading}
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
			<div className="flex flex-wrap items-center gap-2">
				<span className="min-w-24 text-center text-sm text-muted-foreground">
					Page {pagination.pageIndex + 1}
					{pagination.mode === "page" &&
						` of ${Math.max(pagination.pageCount, 1)}`}
				</span>
				{pagination.mode === "page" ? (
					<>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="First page"
							disabled={pagination.pageIndex === 0 || isLoading}
							onClick={onFirstPage}
						>
							<ChevronsLeft aria-hidden="true" />
						</Button>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Previous page"
							disabled={!canPreviousPage || isLoading}
							onClick={onPreviousPage}
						>
							<ChevronLeft aria-hidden="true" />
						</Button>
						<nav
							className="flex items-center gap-1"
							aria-label="Pagination pages"
						>
							{pageItems.map((item) =>
								item.kind === "ellipsis" ? (
									<span
										key={`ellipsis-${item.id}`}
										className="px-1 text-sm text-muted-foreground"
										aria-hidden="true"
									>
										…
									</span>
								) : (
									<Button
										key={item.page}
										variant={
											item.page === pagination.pageIndex ? "default" : "outline"
										}
										size="sm"
										aria-label={`Page ${item.page + 1}`}
										aria-current={
											item.page === pagination.pageIndex ? "page" : undefined
										}
										disabled={isLoading}
										onClick={() => pagination.onPageChange(item.page)}
									>
										{item.page + 1}
									</Button>
								),
							)}
						</nav>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Next page"
							disabled={!canNextPage || isLoading}
							onClick={onNextPage}
						>
							<ChevronRight aria-hidden="true" />
						</Button>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Last page"
							disabled={
								pagination.pageIndex >= pagination.pageCount - 1 || isLoading
							}
							onClick={onLastPage}
						>
							<ChevronsRight aria-hidden="true" />
						</Button>
					</>
				) : (
					<>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Previous page"
							disabled={!pagination.hasPreviousPage || isLoading}
							onClick={pagination.onPreviousPage}
						>
							<ChevronLeft aria-hidden="true" />
						</Button>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label="Next page"
							disabled={!pagination.hasNextPage || isLoading}
							onClick={pagination.onNextPage}
						>
							<ChevronRight aria-hidden="true" />
						</Button>
					</>
				)}
			</div>
		</div>
	);
}
