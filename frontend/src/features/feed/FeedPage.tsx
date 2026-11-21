import { Link } from "@tanstack/react-router";
import { ArrowLeft, Settings2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ErrorState } from "#/components/ErrorState";
import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import { Button, buttonVariants } from "#/components/ui/button";
import { Skeleton } from "#/components/ui/skeleton";
import { useEntries, useToggleEntry } from "#/hooks/queries";
import { FeedFilters } from "./FeedFilters";
import { FeedList } from "./FeedList";
import { FeedSkeleton } from "./FeedSkeleton";
import { useFeedState } from "./FeedState";
export function FeedPage({
	sourceId,
	savedOnly = false,
}: {
	sourceId?: string;
	savedOnly?: boolean;
}) {
	const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");
	const { allSources, sourcesLoading, sourcesError } = useFeedState();
	const source = allSources.find((item) => item.id === sourceId);
	const loadMoreRef = useRef<HTMLDivElement>(null);
	const query = useEntries({
		sourceId,
		savedOnly,
		unread: false,
		sortDirection,
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
			{sourceId ? (
				<header className="relative mb-6 overflow-hidden rounded-xl border border-border bg-card">
					<svg
						aria-hidden="true"
						className="pointer-events-none absolute inset-y-0 right-0 h-full w-3/4 text-primary"
						viewBox="0 0 700 160"
						preserveAspectRatio="none"
					>
						<path
							d="M120 160 C250 90 310 170 420 100 S590 0 700 0 V160 Z"
							fill="currentColor"
							opacity="0.06"
						/>
						<path
							d="M300 160 C400 85 470 130 560 55 S670 20 700 12 V160 Z"
							fill="currentColor"
							opacity="0.09"
						/>
					</svg>
					<div className="relative flex min-h-32 flex-wrap items-center gap-4 p-5 sm:gap-5 sm:p-6">
						{source ? (
							<>
								<SourceAvatar
									name={source.name}
									imageId={source.profileImageId}
									sourceId={source.id}
									size={64}
								/>
								<div className="min-w-0 flex-1">
									<h1 className="break-words text-xl font-bold tracking-tight sm:text-2xl">
										{source.name}
									</h1>
									<div className="mt-2 flex flex-wrap items-center gap-2">
										<SourceTypeBadge type={source.type} />
										{!source.active && (
											<span className="text-xs text-muted-foreground">
												Inactive
											</span>
										)}
									</div>
								</div>
								<Link
									to="/sources"
									className={buttonVariants({
										variant: "outline",
										size: "sm",
										className: "bg-background",
									})}
								>
									<Settings2 data-icon="inline-start" />
									Manage sources
								</Link>
							</>
						) : sourcesLoading ? (
							<>
								<Skeleton className="size-16 shrink-0 rounded-full" />
								<div className="flex flex-col gap-2">
									<Skeleton className="h-6 w-40" />
									<Skeleton className="h-5 w-16" />
								</div>
								<span className="sr-only">Loading source</span>
							</>
						) : (
							<div>
								<h1 className="text-xl font-bold">Source</h1>
								<p className="mt-1 text-sm text-muted-foreground">
									{sourcesError
										? "Could not load source details."
										: "Source details are unavailable."}
								</p>
							</div>
						)}
					</div>
				</header>
			) : (
				<h1 className="sr-only">{savedOnly ? "Saved" : "My Feed"}</h1>
			)}
			<FeedFilters
				sortDirection={sortDirection}
				setSortDirection={setSortDirection}
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
	);
}
