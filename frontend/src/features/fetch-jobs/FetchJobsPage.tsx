import { useState } from "react";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { Button } from "#/components/ui/button";
import { Skeleton } from "#/components/ui/skeleton";
import { useFeedState } from "#/features/feed/FeedState";
import type { Source } from "#/features/feed/types";
import { useCreateFetchJob, useFetchJob } from "#/hooks/queries";

const exampleHistory = [
	{
		id: "example-succeeded",
		source: "Sample RSS",
		status: "Succeeded",
		entries: 3,
	},
	{ id: "example-failed", source: "Sample RSS", status: "Failed", entries: 0 },
];

export function FetchJobsPage() {
	const {
		allSources,
		sourcesLoading,
		sourcesError,
		sourcesErrorValue,
		refetchSources,
	} = useFeedState();
	const fetchableSources = allSources.filter(
		(source: Source) => source.active && source.type !== "manual",
	);
	const [sourceId, setSourceId] = useState("");
	const [jobId, setJobId] = useState<string | null>(null);
	const createJob = useCreateFetchJob();
	const jobQuery = useFetchJob(jobId);
	const job = jobQuery.data;
	const terminal = job?.status === "succeeded" || job?.status === "failed";

	return (
		<div className="space-y-10">
			<header>
				<p className="text-xs font-bold uppercase tracking-widest text-primary">
					Tools
				</p>
				<h1 className="mt-2 text-3xl font-bold tracking-tight">Fetch Jobs</h1>
				<p className="mt-2 text-muted-foreground">
					Start a source fetch and track the latest job status
				</p>
			</header>

			<section
				className="rounded-2xl border bg-card p-6"
				aria-labelledby="start-fetch-title"
			>
				<h2 id="start-fetch-title" className="text-lg font-semibold">
					Start Fetch
				</h2>
				<div className="mt-4 flex flex-col gap-3 sm:flex-row">
					<label className="flex-1 text-sm font-medium" htmlFor="fetch-source">
						Source to fetch
						<select
							id="fetch-source"
							className="mt-2 block h-11 w-full rounded-lg border bg-background px-3"
							value={sourceId}
							onChange={(event) => setSourceId(event.target.value)}
							disabled={sourcesLoading || fetchableSources.length === 0}
						>
							<option value="">Select a source</option>
							{fetchableSources.map((source) => (
								<option key={source.id} value={source.id}>
									{source.name}
								</option>
							))}
						</select>
					</label>
					<button
						type="button"
						className="mt-auto h-11 rounded-lg bg-primary px-5 font-semibold text-primary-foreground disabled:opacity-50"
						disabled={
							!sourceId || createJob.isPending || Boolean(jobId && !terminal)
						}
						onClick={() =>
							createJob.mutate(sourceId, {
								onSuccess: (job) => setJobId(job.id),
							})
						}
					>
						{createJob.isPending ? "Starting..." : "Start Fetch"}
					</button>
				</div>
				{sourcesLoading && (
					<output className="mt-3 space-y-2" aria-busy="true">
						<span className="sr-only">Loading sources</span>
						<Skeleton className="h-3 w-28" />
						<Skeleton className="h-11 w-full rounded-lg" />
					</output>
				)}
				{sourcesError && (
					<ErrorState
						size="compact"
						error={sourcesErrorValue ?? "Sources unavailable"}
						message="Could not load sources. Please try again"
						action={
							<Button onClick={() => void refetchSources()}>Try again</Button>
						}
					/>
				)}
				{!sourcesLoading && !sourcesError && fetchableSources.length === 0 && (
					<NoDataState
						size="compact"
						illustration="feed"
						title="No sources available to fetch"
						description="Add or activate an RSS source before fetching"
					/>
				)}
				{createJob.isError && (
					<p role="alert" className="mt-3 text-sm text-destructive">
						Could not start fetch. Please try again
					</p>
				)}
			</section>

			<section
				className="rounded-2xl border bg-card p-6"
				aria-labelledby="current-job-title"
			>
				<h2 id="current-job-title" className="text-lg font-semibold">
					Latest Fetch Job
				</h2>
				{!jobId && (
					<NoDataState
						size="compact"
						illustration="sleeping"
						title="No fetch jobs yet"
						description="Select a source and start a fetch to see its status here"
					/>
				)}
				{jobQuery.isLoading && (
					<output className="mt-3 space-y-2" aria-busy="true">
						<span className="sr-only">Loading fetch job status</span>
						<Skeleton className="h-4 w-48" />
						<Skeleton className="h-4 w-36" />
					</output>
				)}
				{jobQuery.isError && (
					<ErrorState
						size="compact"
						error={jobQuery.error}
						message="Could not load fetch job status"
						action={
							<Button onClick={() => void jobQuery.refetch()}>Try again</Button>
						}
					/>
				)}
				{job && (
					<div className="mt-3 space-y-1 text-sm">
						<p>
							Status:{" "}
							<strong>
								{job.status === "pending"
									? "Pending"
									: job.status === "running"
										? "In progress"
										: job.status === "succeeded"
											? "Succeeded"
											: "Failed"}
							</strong>
						</p>
						{job.status === "succeeded" && (
							<p>New entries found: {job.entriesCreated}</p>
						)}
						{job.status === "failed" && (
							<p role="alert" className="text-destructive">
								Failed after {job.attempt} of {job.maxAttempts} attempts. The API
								did not return error details
							</p>
						)}
						{!terminal && (
							<p className="text-muted-foreground">Tracking status...</p>
						)}
					</div>
				)}
			</section>

			<section
				className="rounded-2xl border bg-card p-6"
				aria-labelledby="example-history-title"
			>
				<h2 id="example-history-title" className="text-lg font-semibold">
					Sample History
				</h2>
				<p className="mt-1 text-sm text-muted-foreground">
					Sample data for display only; this is not actual fetch history
				</p>
				<ul className="mt-4 divide-y">
					{exampleHistory.map((item) => (
						<li key={item.id} className="flex justify-between py-3 text-sm">
							<span>{item.source}</span>
							<span>
								{item.status} · {item.entries} Entries
							</span>
						</li>
					))}
				</ul>
			</section>
		</div>
	);
}
