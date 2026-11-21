import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";

export function FeedFilters({
	sortDirection,
	setSortDirection,
}: {
	sortDirection: "asc" | "desc";
	setSortDirection: (direction: "asc" | "desc") => void;
}) {
	return (
		<div className="mb-4 flex flex-wrap items-center justify-end gap-2 py-2">
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
