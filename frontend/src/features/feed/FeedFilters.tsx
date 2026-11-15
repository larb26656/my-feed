import { Button } from "#/components/ui/button";
import { type FeedFilter, filters } from "./data";

export function FeedFilters({
	filter,
	setFilter,
}: {
	filter: FeedFilter;
	setFilter: (filter: FeedFilter) => void;
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
			</div>
			<span className="text-xs text-muted-foreground">ล่าสุดก่อน ↓</span>
		</div>
	);
}
