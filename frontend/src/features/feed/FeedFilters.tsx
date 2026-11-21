import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { Button, buttonVariants } from "#/components/ui/button";
import { type FeedFilter, filters } from "./types";

export function FeedFilters({
	filter,
	setFilter,
	sortDirection,
	setSortDirection,
}: {
	filter: FeedFilter;
	setFilter: (filter: FeedFilter) => void;
	sortDirection: "asc" | "desc";
	setSortDirection: (direction: "asc" | "desc") => void;
}) {
	return (
		<div className="mb-4 flex flex-wrap items-center justify-between gap-2 py-2">
			<div className="flex flex-wrap items-center gap-1">
				{filters.map((item) => (
					<Button
						key={item}
						variant={filter === item ? "secondary" : "ghost"}
						size="sm"
						className={
							filter === item
								? "bg-primary/15 text-primary hover:bg-primary/20"
								: ""
						}
						aria-pressed={filter === item}
						onClick={() => setFilter(item)}
					>
						{item}
					</Button>
				))}
				<Link
					to="/saved"
					className={buttonVariants({ variant: "outline", size: "sm" })}
				>
					Saved
				</Link>
			</div>
			<div className="flex items-center gap-3">
				<select
					aria-label="Sort entries"
					value={sortDirection}
					onChange={(event) =>
						setSortDirection(event.target.value as "asc" | "desc")
					}
					className="min-h-10 rounded-md bg-transparent px-2 text-xs text-muted-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-primary"
				>
					<option value="desc">Newest first</option>
					<option value="asc">Oldest first</option>
				</select>
				<div className="h-6 border-l border-border/70" aria-hidden="true" />
				<Link
					to="/record-link"
					aria-label="Save Link"
					className={buttonVariants({ variant: "ghost", size: "icon-sm" })}
				>
					<Plus />
				</Link>
			</div>
		</div>
	);
}
