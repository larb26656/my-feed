import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";

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
			<div className="flex items-center gap-2 text-sm text-muted-foreground">
				<span>Rows per page</span>
				<Select
					value={String(pageSize)}
					disabled={isLoading}
					onValueChange={(value) => {
						if (value !== null) onPageSizeChange(Number(value) as TPageSize);
					}}
				>
					<SelectTrigger aria-label="Rows per page">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectGroup>
							{pageSizeOptions.map((size) => (
								<SelectItem key={size} value={String(size)}>
									{size}
								</SelectItem>
							))}
						</SelectGroup>
					</SelectContent>
				</Select>
			</div>
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
