import { ErrorState } from "#/components/ErrorState";
import { InfiniteScroll } from "#/components/InfiniteScroll";
import { PullToRefresh } from "#/components/PullToRefresh";
import { Button } from "#/components/ui/button";
import { type useEntries, useToggleEntry } from "#/hooks/queries";
import { FeedList } from "./FeedList";
import { FeedSkeleton } from "./FeedSkeleton";
import { useFeedState } from "./FeedState";

export function FeedEntries({
	query,
	children,
}: {
	query: ReturnType<typeof useEntries>;
	children?: React.ReactNode;
}) {
	const { allSources } = useFeedState();
	const mutation = useToggleEntry();
	const visible = query.data?.pages.flatMap((page) => page.data) ?? [];
	return (
		<PullToRefresh onRefresh={() => query.refetch()}>
			<section className="mx-auto max-w-6xl md:px-5 md:py-7">
				{children}

				<InfiniteScroll
					next={{
						hasPage: query.hasNextPage,
						isLoading: query.isFetchingNextPage,
						isError: query.isFetchNextPageError,
						load: query.fetchNextPage,
					}}
				>
					{query.isLoading ? (
						<FeedSkeleton />
					) : query.isError ? (
						<ErrorState
							error={query.error}
							message="Could not load entries. Please try again"
							action={
								<Button onClick={() => void query.refetch()}>Try again</Button>
							}
						/>
					) : (
						<FeedList
							visible={visible}
							sources={allSources}
							disabled={mutation.isPending}
							onToggle={(entry, action) => mutation.mutate({ entry, action })}
						/>
					)}
					{mutation.isError && <p role="alert">Could not update the entry</p>}
				</InfiniteScroll>
			</section>
		</PullToRefresh>
	);
}
