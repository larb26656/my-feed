import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { NoDataState } from "#/components/NoDataState";
import { PullToRefresh } from "#/components/PullToRefresh";
import { buttonVariants } from "#/components/ui/button";
import { useSourceActions } from "#/features/sources/useSourceActions";
import { useEntries } from "#/hooks/queries";
import { FeedEntries } from "./FeedEntries";
import { FeedSort } from "./FeedSort";
import { useFeedState } from "./FeedState";
import { SourceHeader } from "./SourceHeader";
import { useFeedSort } from "./useFeedSort";

export function SourceFeedPage({ sourceId }: { sourceId: string }) {
	const { allSources, sourcesLoading, sourcesError } = useFeedState();
	const source = allSources.find((item) => item.id === sourceId);
	const { actions, dialogs } = useSourceActions();
	const sort = useFeedSort(
		source?.type === "manual" ? "createdAtSort" : "publishedAtSort",
	);
	const query = useEntries({
		sourceId,
		sortField: sort.sortField,
		sortDirection: sort.sortDirection,
	});
	return (
		<PullToRefresh onRefresh={() => query.refetch()}>
			<section className="mx-auto max-w-6xl md:px-5 md:py-7">
				<Link
					to="/sources"
					className={buttonVariants({
						variant: "ghost",
						className: "ml-1 mt-2 mb-2 md:-ml-3 md:mt-0",
					})}
				>
					<ArrowLeft data-icon="inline-start" />
					All Sources
				</Link>
				<SourceHeader
					source={source}
					loading={sourcesLoading}
					error={sourcesError}
					actions={actions}
				/>
				{dialogs}
				<div className="mb-2 flex items-center justify-end px-4 py-1 md:px-0">
					<FeedSort {...sort} />
				</div>
				<FeedEntries
					query={query}
					emptyState={
						<NoDataState
							illustration="feed"
							title="Nothing to read yet"
							description="This source has no entries yet"
						/>
					}
				/>
			</section>
		</PullToRefresh>
	);
}
