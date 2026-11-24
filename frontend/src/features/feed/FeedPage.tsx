import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ErrorState } from "#/components/ErrorState";
import { PullToRefresh } from "#/components/PullToRefresh";
import { Button, buttonVariants } from "#/components/ui/button";
import { useEntries, useToggleEntry } from "#/hooks/queries";
import { FeedFilters } from "./FeedFilters";
import { FeedList } from "./FeedList";
import { FeedSkeleton } from "./FeedSkeleton";
import { useFeedState } from "./FeedState";
import { SourceHeader } from "./SourceHeader";
export function FeedPage({
	sourceId,
	savedOnly = false,
}: {
	sourceId?: string;
	savedOnly?: boolean;
}) {
	const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");
	const [chosenSortField, setSortField] = useState<
		"publishedAtSort" | "createdAtSort" | "savedAtSort" | null
	>(null);
	const { allSources, sourcesLoading, sourcesError } = useFeedState();
	const source = allSources.find((item) => item.id === sourceId);
	const sortField =
		chosenSortField ??
		(savedOnly
			? "savedAtSort"
			: source?.type === "manual"
				? "createdAtSort"
				: "publishedAtSort");
	const loadMoreRef = useRef<HTMLDivElement>(null);
	const query = useEntries({
		sourceId,
		savedOnly,
		unread: false,
		sortDirection,
		sortField,
	});
	const mutation = useToggleEntry();
	const visible = query.data?.pages.flatMap((page) => page.data) ?? [];
	useEffect(() => {
		const target = loadMoreRef.current;
		if (
			!target ||
			!query.hasNextPage ||
			query.isFetchingNextPage ||
			query.isFetchNextPageError
		) {
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
	}, [
		query.fetchNextPage,
		query.hasNextPage,
		query.isFetchingNextPage,
		query.isFetchNextPageError,
	]);
	return (
		<PullToRefresh onRefresh={() => query.refetch()}>
			<section className="mx-auto max-w-6xl md:px-5 md:py-7">
				{sourceId && (
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
				)}
				{sourceId ? (
					<SourceHeader
						source={source}
						loading={sourcesLoading}
						error={sourcesError}
					/>
				) : (
					<h1 className="sr-only">{savedOnly ? "Saved" : "My Feed"}</h1>
				)}
				<FeedFilters
					sortField={sortField}
					setSortField={setSortField}
					sortDirection={sortDirection}
					setSortDirection={setSortDirection}
					className={sourceId ? "mb-2 py-1" : undefined}
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
					<div
						ref={loadMoreRef}
						className="flex min-h-12 items-center justify-center"
					>
						{query.isFetchingNextPage && <output>Loading...</output>}
						{query.isFetchNextPageError && (
							<div className="flex items-center gap-3">
								<p role="alert">Could not load more entries</p>
								<Button onClick={() => void query.fetchNextPage()}>
									Try again
								</Button>
							</div>
						)}
					</div>
				)}
			</section>
		</PullToRefresh>
	);
}
