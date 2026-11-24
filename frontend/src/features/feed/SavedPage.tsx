import { useEntries } from "#/hooks/queries";
import { FeedEntries } from "./FeedEntries";
import { FeedFilters } from "./FeedFilters";
import { useFeedSort } from "./useFeedSort";

export function SavedPage() {
	const sort = useFeedSort("savedAtSort");
	const query = useEntries({
		savedOnly: true,
		sortField: sort.sortField,
		sortDirection: sort.sortDirection,
	});
	return (
		<FeedEntries query={query}>
			<h1 className="sr-only">Saved</h1>
			<FeedFilters {...sort} />
		</FeedEntries>
	);
}
