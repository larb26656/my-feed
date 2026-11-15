import { Button } from "#/components/ui/button";
import { type FeedFilter, filters } from "./types";

export function FeedFilters({
	filter,
	setFilter,
	savedOnly,
	setSavedOnly,
}: {
	filter: FeedFilter;
	setFilter: (filter: FeedFilter) => void;
	savedOnly: boolean;
	setSavedOnly: (saved: boolean) => void;
}) {
	return (
		<div className="flex items-center justify-between border-b pb-1">
			<div className="flex gap-1">
				{filters.map((item) => (
					<Button
						key={item}
						variant={filter === item ? "secondary" : "ghost"}
						size="sm"
						aria-pressed={filter === item}
						onClick={() => setFilter(item)}
					>
						{item}
					</Button>
				))}
				<Button
					variant={savedOnly ? "secondary" : "ghost"}
					size="sm"
					aria-pressed={savedOnly}
					onClick={() => setSavedOnly(!savedOnly)}
				>
					Saved
				</Button>
			</div>
			<span className="text-xs text-muted-foreground">Newest first ↓</span>
		</div>
	);
}
