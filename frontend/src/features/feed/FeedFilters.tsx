import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { Button, buttonVariants } from "#/components/ui/button";
import { type FeedFilter, filters } from "./types";

export function FeedFilters({
	filter,
	setFilter,
}: {
	filter: FeedFilter;
	setFilter: (filter: FeedFilter) => void;
}) {
	return (
		<div className="flex items-center justify-between px-3 py-2">
			<div className="flex items-center gap-1">
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
			</div>
			<div className="flex items-center gap-3">
				<span className="text-xs text-muted-foreground">Newest first ↓</span>
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
