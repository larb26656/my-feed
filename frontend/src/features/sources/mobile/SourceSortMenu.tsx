import { ArrowUpDown } from "lucide-react";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import type {
	SourceSort,
	SourceSortDirection,
	SourceSortField,
} from "#/features/sources/source.api";

const options: {
	sortBy: SourceSortField;
	sortDirection: SourceSortDirection;
	label: string;
}[] = [
	{ sortBy: "name", sortDirection: "asc", label: "Name (A-Z)" },
	{ sortBy: "name", sortDirection: "desc", label: "Name (Z-A)" },
	{ sortBy: "type", sortDirection: "asc", label: "Type (A-Z)" },
	{ sortBy: "type", sortDirection: "desc", label: "Type (Z-A)" },
	{ sortBy: "createdAt", sortDirection: "desc", label: "Date added (newest)" },
	{ sortBy: "createdAt", sortDirection: "asc", label: "Date added (oldest)" },
];

export function sourceSortLabel(sort: SourceSort) {
	return (
		options.find(
			(option) =>
				option.sortBy === sort.sortBy &&
				option.sortDirection === sort.sortDirection,
		)?.label ?? "Name (A-Z)"
	);
}

export function formatSourceCount(total: number) {
	return `${total} ${total === 1 ? "source" : "sources"}`;
}

const menuItemClass =
	"flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted aria-checked:bg-muted aria-checked:font-medium";

export function SourceSortMenu({
	sort,
	onSortChange,
}: {
	sort: SourceSort;
	onSortChange: (sort: SourceSort) => void;
}) {
	const [open, setOpen] = useState(false);

	return (
		<div className="relative">
			<Button
				type="button"
				variant="outline"
				size="icon"
				className="size-10"
				aria-label="Sort sources"
				aria-haspopup="menu"
				aria-expanded={open}
				onClick={() => setOpen((current) => !current)}
			>
				<ArrowUpDown />
			</Button>
			{open && (
				<div
					role="menu"
					aria-label="Sort sources"
					className="absolute right-0 top-full z-30 mt-2 min-w-52 rounded-xl border bg-popover p-1 shadow-lg"
				>
					{options.map((option) => {
						const selected =
							option.sortBy === sort.sortBy &&
							option.sortDirection === sort.sortDirection;
						return (
							<button
								key={option.label}
								type="button"
								role="menuitemradio"
								aria-checked={selected}
								className={menuItemClass}
								onClick={() => {
									setOpen(false);
									onSortChange({
										sortBy: option.sortBy,
										sortDirection: option.sortDirection,
									});
								}}
							>
								{option.label}
							</button>
						);
					})}
				</div>
			)}
		</div>
	);
}
