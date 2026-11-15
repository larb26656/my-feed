import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ErrorState } from "#/components/ErrorState";
import { Button, buttonVariants } from "#/components/ui/button";
import { useEntries, useToggleEntry } from "#/hooks/queries";
import { FeedFilters } from "./FeedFilters";
import { FeedList } from "./FeedList";
import { FeedSkeleton } from "./FeedSkeleton";
import { useFeedState } from "./FeedState";
import type { FeedFilter } from "./types";
export function FeedPage({
	sourceId,
	savedOnly = false,
}: {
	sourceId?: string;
	savedOnly?: boolean;
}) {
	const [filter, setFilter] = useState<FeedFilter>("All");
	const { allSources } = useFeedState();
	const loadMoreRef = useRef<HTMLDivElement>(null);
	const query = useEntries({
		sourceId,
		savedOnly,
		unread: filter === "Unread",
	});
	const mutation = useToggleEntry();
	const visible = query.data?.pages.flatMap((page) => page.data) ?? [];
	useEffect(() => {
		const target = loadMoreRef.current;
		if (!target || !query.hasNextPage || query.isFetchingNextPage || query.isFetchNextPageError) {
			return;
		}

		const observer = new IntersectionObserver(
			([entry]) => {
				if (entry.isIntersecting) void query.fetchNextPage();
			},
			{ rootMargin: "200px" },
		);
		observer.observe(target);
		return () => observer.disconnect();
	}, [query.fetchNextPage, query.hasNextPage, query.isFetchingNextPage, query.isFetchNextPageError]);
	return (
		<section>
			{sourceId && (
				<Link
					to="/sources"
					className={buttonVariants({
						variant: "ghost",
						className: "-ml-3 mb-4",
					})}
				>
					<ArrowLeft data-icon="inline-start" />
					All Sources
				</Link>
			)}
			<h1 className="sr-only">
				{allSources.find((source) => source.id === sourceId)?.name ??
					(savedOnly ? "Saved" : "My Feed")}
			</h1>
			<FeedFilters
				filter={filter}
				setFilter={setFilter}
			/>
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
			{query.hasNextPage && (
				<div ref={loadMoreRef} className="flex min-h-12 items-center justify-center">
					{query.isFetchingNextPage && <p role="status">Loading...</p>}
					{query.isFetchNextPageError && (
						<div className="flex items-center gap-3">
							<p role="alert">Could not load more entries</p>
							<Button onClick={() => void query.fetchNextPage()}>Try again</Button>
						</div>
					)}
				</div>
			)}
		</section>
	);
}
