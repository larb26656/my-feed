import { ArrowUpDown } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuRadioGroup,
	DropdownMenuRadioItem,
	DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
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

export function SourceSortMenu({
	sort,
	onSortChange,
}: {
	sort: SourceSort;
	onSortChange: (sort: SourceSort) => void;
}) {
	const value = `${sort.sortBy}:${sort.sortDirection}`;
	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				render={
					<Button
						type="button"
						variant="outline"
						size="icon"
						className="size-10"
						aria-label="Sort sources"
					/>
				}
			>
				<ArrowUpDown />
			</DropdownMenuTrigger>
			<DropdownMenuContent
				align="end"
				className="min-w-52"
				aria-label="Sort sources"
			>
				<DropdownMenuGroup>
					<DropdownMenuRadioGroup
						value={value}
						onValueChange={(next) => {
							const option = options.find(
								(item) => `${item.sortBy}:${item.sortDirection}` === next,
							);
							if (option)
								onSortChange({
									sortBy: option.sortBy,
									sortDirection: option.sortDirection,
								});
						}}
					>
						{options.map((option) => (
							<DropdownMenuRadioItem
								key={option.label}
								value={`${option.sortBy}:${option.sortDirection}`}
							>
								{option.label}
							</DropdownMenuRadioItem>
						))}
					</DropdownMenuRadioGroup>
				</DropdownMenuGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
