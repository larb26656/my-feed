import { NoDataState } from "#/components/NoDataState";
import { PullToRefresh } from "#/components/PullToRefresh";
import { useEntries } from "#/hooks/queries";
import { FeedEntries } from "./FeedEntries";
import { FeedSort } from "./FeedSort";
import { useFeedSort } from "./useFeedSort";

export function SavedPage() {
	const sort = useFeedSort("savedAtSort");
	const query = useEntries({
		savedOnly: true,
		sortField: sort.sortField,
		sortDirection: sort.sortDirection,
	});
	return (
		<PullToRefresh onRefresh={() => query.refetch()}>
			<section className="mx-auto max-w-6xl md:px-5 md:py-7">
				<h1 className="sr-only">Saved</h1>
				<div className="mb-4 flex items-center justify-end px-4 py-2 md:px-0">
					<FeedSort {...sort} />
				</div>
				<FeedEntries
					query={query}
					emptyState={
						<NoDataState
							illustration="feed"
							title="No saved entries yet"
							description="Bookmark entries to find them here"
						/>
					}
				/>
			</section>
		</PullToRefresh>
	);
}
