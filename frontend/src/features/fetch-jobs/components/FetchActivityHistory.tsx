import { Search } from "lucide-react";
import { CursorPaginationControls } from "#/components/CursorPaginationControls";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { Button } from "#/components/ui/button";
import { Card, CardContent } from "#/components/ui/card";
import { Input } from "#/components/ui/input";
import { Skeleton } from "#/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";
import type { Source } from "#/features/feed/types";
import type { FetchJob } from "../types";
import { ActiveRunRow } from "./ActiveRunRow";
import { FetchHistoryRow } from "./FetchHistoryRow";

type ActivityTab = "all" | "succeeded" | "failed" | "in-progress";

export function FetchActivityHistory<TPageSize extends number>({
	jobs,
	activeJobs,
	sources,
	tab,
	onTabChange,
	search,
	onSearchChange,
	isLoading,
	error,
	onRetry,
	pageSize,
	pageSizeOptions,
	onPageSizeChange,
	hasPreviousPage,
	hasNextPage,
	isLoadingPage,
	onPreviousPage,
	onNextPage,
}: {
	jobs: FetchJob[];
	activeJobs: FetchJob[];
	sources: Source[];
	tab: ActivityTab;
	onTabChange: (tab: ActivityTab) => void;
	search: string;
	onSearchChange: (search: string) => void;
	isLoading: boolean;
	error?: Error | null;
	onRetry: () => void;
	pageSize: TPageSize;
	pageSizeOptions: readonly TPageSize[];
	onPageSizeChange: (pageSize: TPageSize) => void;
	hasPreviousPage: boolean;
	hasNextPage: boolean;
	isLoadingPage: boolean;
	onPreviousPage: () => void;
	onNextPage: () => void;
}) {
	const sourceById = new Map(sources.map((source) => [source.id, source]));
	const visibleJobs = (tab === "in-progress" ? activeJobs : jobs).filter(
		(job) => {
			if (tab === "succeeded" && job.status !== "succeeded") return false;
			if (tab === "failed" && job.status !== "failed") return false;
			const source = sourceById.get(job.sourceId);
			const value =
				`${source?.name ?? ""} ${source?.rssUrl ?? ""}`.toLowerCase();
			return value.includes(search.trim().toLowerCase());
		},
	);

	const content = error ? (
		<ErrorState
			size="compact"
			error={error}
			message="Could not load recent activity"
			action={<Button onClick={onRetry}>Try again</Button>}
		/>
	) : isLoading ? (
		<div className="space-y-4" aria-busy="true">
			{[0, 1, 2].map((row) => (
				<div key={row} className="flex items-center gap-4">
					<Skeleton className="size-10 rounded-full" />
					<div className="flex-1 space-y-2">
						<Skeleton className="h-4 w-40" />
						<Skeleton className="h-4 w-28" />
					</div>
				</div>
			))}
		</div>
	) : visibleJobs.length === 0 ? (
		<NoDataState
			size="compact"
			illustration="sleeping"
			title={search ? "No matching runs" : "No fetch history yet"}
			description={
				search
					? "Try a different source name."
					: "Fetch runs will appear here after a source is checked."
			}
		/>
	) : (
		<div className="divide-y">
			{visibleJobs.map((job) =>
				tab === "in-progress" ? (
					<ActiveRunRow
						key={job.id}
						job={job}
						source={sourceById.get(job.sourceId)}
					/>
				) : (
					<FetchHistoryRow
						key={job.id}
						job={job}
						source={sourceById.get(job.sourceId)}
					/>
				),
			)}
		</div>
	);

	return (
		<section aria-labelledby="recent-activity-title" className="space-y-4">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<h2
					id="recent-activity-title"
					className="text-xl font-semibold tracking-tight"
				>
					Recent activity
				</h2>
				<div className="relative w-full sm:w-64">
					<Search
						className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
						aria-hidden="true"
					/>
					<Input
						value={search}
						onChange={(event) => onSearchChange(event.target.value)}
						placeholder="Find a source..."
						aria-label="Find a source"
						className="pl-9"
					/>
				</div>
			</div>
			<Tabs
				value={tab}
				onValueChange={(value) => onTabChange(value as ActivityTab)}
			>
				<TabsList variant="line">
					<TabsTrigger value="all">All runs</TabsTrigger>
					<TabsTrigger value="succeeded">Succeeded</TabsTrigger>
					<TabsTrigger value="failed">Failed</TabsTrigger>
					<TabsTrigger value="in-progress">In progress</TabsTrigger>
				</TabsList>
				{(["all", "succeeded", "failed", "in-progress"] as ActivityTab[]).map(
					(value) => (
						<TabsContent key={value} value={value}>
							<Card>
								<CardContent>
									{content}
									{!error && value === tab && value !== "in-progress" && (
										<CursorPaginationControls
											pageSize={pageSize}
											pageSizeOptions={pageSizeOptions}
											onPageSizeChange={onPageSizeChange}
											hasPreviousPage={hasPreviousPage}
											hasNextPage={hasNextPage}
											isLoading={isLoadingPage}
											onPrevious={onPreviousPage}
											onNext={onNextPage}
										/>
									)}
								</CardContent>
							</Card>
						</TabsContent>
					),
				)}
			</Tabs>
		</section>
	);
}
