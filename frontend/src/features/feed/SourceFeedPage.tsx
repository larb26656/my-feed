import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";
import { useSourceActions } from "#/features/sources/useSourceActions";
import { useEntries } from "#/hooks/queries";
import { FeedEntries } from "./FeedEntries";
import { FeedFilters } from "./FeedFilters";
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
		<FeedEntries query={query}>
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
			<FeedFilters {...sort} className="mb-2 py-1" />
		</FeedEntries>
	);
}
