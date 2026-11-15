import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "#/components/ui/button";

type CursorPaginationControlsProps<TPageSize extends number> = {
	pageSize: TPageSize;
	pageSizeOptions: readonly TPageSize[];
	onPageSizeChange: (pageSize: TPageSize) => void;
	hasPreviousPage: boolean;
	hasNextPage: boolean;
	isLoading?: boolean;
	onPrevious: () => void;
	onNext: () => void;
};

export function CursorPaginationControls<TPageSize extends number>({
	pageSize,
	pageSizeOptions,
	onPageSizeChange,
	hasPreviousPage,
	hasNextPage,
	isLoading = false,
	onPrevious,
	onNext,
}: CursorPaginationControlsProps<TPageSize>) {
	return (
		<div className="flex items-center justify-end gap-2 border-t pt-3">
			<label className="mr-1 flex items-center gap-2 text-sm text-muted-foreground">
				<span>Rows per page</span>
				<select
					className="h-8 rounded-lg border bg-background px-2 text-foreground"
					value={pageSize}
					disabled={isLoading}
					onChange={(event) =>
						onPageSizeChange(Number(event.target.value) as TPageSize)
					}
				>
					{pageSizeOptions.map((size) => (
						<option key={size} value={size}>
							{size}
						</option>
					))}
				</select>
			</label>
			<Button
				variant="ghost"
				size="icon-sm"
				aria-label="Previous page"
				disabled={!hasPreviousPage || isLoading}
				onClick={onPrevious}
			>
				<ChevronLeft aria-hidden="true" />
			</Button>
			<Button
				variant="ghost"
				size="icon-sm"
				aria-label="Next page"
				disabled={!hasNextPage || isLoading}
				onClick={onNext}
			>
				<ChevronRight aria-hidden="true" />
			</Button>
		</div>
	);
}
