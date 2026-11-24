import { useEntries } from "#/hooks/queries";
import { FeedEntries } from "./FeedEntries";
import { FeedFilters } from "./FeedFilters";
import { useFeedSort } from "./useFeedSort";

export function FeedPage() {
	const sort = useFeedSort("publishedAtSort");
	const query = useEntries({
		sortField: sort.sortField,
		sortDirection: sort.sortDirection,
	});
	return (
		<FeedEntries query={query}>
			<h1 className="sr-only">My Feed</h1>
			<FeedFilters {...sort} />
		</FeedEntries>
	);
}
