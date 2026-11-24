import { LoaderCircle } from "lucide-react";
import { useEffect, useRef } from "react";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { Button } from "#/components/ui/button";
import { Skeleton } from "#/components/ui/skeleton";
import type { SourceActionHandlers } from "#/features/sources/SourceActions";
import {
	type SourceFilters,
	type SourceSort,
	sourcesEmptyMessage,
} from "#/features/sources/source.api";
import { SourceCard } from "./SourceCard";
import { formatSourceCount, sourceSortLabel } from "./SourceSortMenu";
import { useInfiniteSources } from "./useInfiniteSources";

function SourceCardSkeleton() {
	return (
		<div className="flex items-center gap-3 rounded-xl border bg-card px-3 py-3">
			<Skeleton className="size-10 rounded-full" />
			<div className="flex-1 space-y-2">
				<Skeleton className="h-4 w-32" />
				<Skeleton className="h-5 w-20 rounded-full" />
			</div>
		</div>
	);
}

export function SourceListView({
	active,
	filters,
	sort,
	actions,
}: {
	active: boolean;
	filters: SourceFilters;
	sort: SourceSort;
	actions: SourceActionHandlers;
}) {
	const query = useInfiniteSources(active, filters, sort);
	const sources = query.data?.pages.flatMap((page) => page.data) ?? [];
	const total = query.data?.pages[0]?.meta.total ?? sources.length;
	const label = active ? "active" : "inactive";
	const sentinelRef = useRef<HTMLDivElement>(null);
	const loadingMore = useRef(false);
	const {
		hasNextPage,
		isFetchingNextPage,
		isFetchNextPageError,
		fetchNextPage,
	} = query;

	useEffect(() => {
		if (!isFetchingNextPage) loadingMore.current = false;
	}, [isFetchingNextPage]);

	useEffect(() => {
		const node = sentinelRef.current;
		if (
			!node ||
			!hasNextPage ||
			isFetchingNextPage ||
			isFetchNextPageError ||
			typeof IntersectionObserver === "undefined"
		) {
			return;
		}
		const observer = new IntersectionObserver(
			(entries) => {
				if (!entries.some((entry) => entry.isIntersecting)) return;
				if (loadingMore.current) return;
				loadingMore.current = true;
				void fetchNextPage();
			},
			{ rootMargin: "240px" },
		);
		observer.observe(node);
		return () => observer.disconnect();
	}, [fetchNextPage, hasNextPage, isFetchNextPageError, isFetchingNextPage]);

	return (
		<div className="mt-3" aria-busy={query.isFetching || isFetchingNextPage}>
			{query.data && (
				<div className="mb-3 flex items-center justify-between gap-3 px-1 text-sm text-muted-foreground">
					<p>{formatSourceCount(total)}</p>
					<p>{sourceSortLabel(sort)}</p>
				</div>
			)}

			{query.isLoading ? (
				<div className="space-y-2">
					<p className="sr-only">Loading {label} sources</p>
					<SourceCardSkeleton />
					<SourceCardSkeleton />
					<SourceCardSkeleton />
					<SourceCardSkeleton />
				</div>
			) : query.isError && sources.length === 0 ? (
				<ErrorState
					error={query.error ?? "Sources unavailable"}
					message="Could not load sources. Please try again"
					action={
						<Button onClick={() => void query.refetch()}>Try again</Button>
					}
				/>
			) : sources.length === 0 ? (
				<NoDataState title={sourcesEmptyMessage(active, filters)} />
			) : (
				<ul aria-label={`${label} sources`} className="space-y-2">
					{sources.map((source) => (
						<SourceCard key={source.id} source={source} actions={actions} />
					))}
				</ul>
			)}

			{hasNextPage && (
				<div ref={sentinelRef} className="h-px" aria-hidden="true" />
			)}
			{isFetchingNextPage && (
				<output className="flex items-center justify-center gap-2 py-4 text-sm text-muted-foreground">
					<LoaderCircle className="animate-spin" />
					Loading more
				</output>
			)}
			{isFetchNextPageError && (
				<div className="py-3 text-center">
					<p role="alert" className="text-sm text-destructive">
						Could not load more sources.
					</p>
					<Button
						variant="outline"
						className="mt-2"
						onClick={() => void fetchNextPage()}
					>
						Try again
					</Button>
				</div>
			)}
		</div>
	);
}
