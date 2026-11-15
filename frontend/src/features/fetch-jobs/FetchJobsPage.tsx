import { useEffect, useMemo, useRef, useState } from "react";
import { Play } from "lucide-react";
import { Button } from "#/components/ui/button";
import { useFeedState } from "#/features/feed/FeedState";
import type { FetchJob } from "#/features/fetch-jobs/types";
import {
	useActiveFetchJobs,
	useFetchJobHistory,
	useStartFetchJobs,
} from "#/hooks/queries";
import { useQueryClient } from "@tanstack/react-query";
import { fetchJobKeys } from "#/lib/query-keys";
import { ActiveRunsCard } from "./components/ActiveRunsCard";
import { FetchActivityHistory } from "./components/FetchActivityHistory";

type ActivityTab = "all" | "succeeded" | "failed" | "in-progress";

export function FetchJobsPage() {
	const { allSources } = useFeedState();
	const activeQuery = useActiveFetchJobs();
	const allHistory = useFetchJobHistory();
	const succeededHistory = useFetchJobHistory("succeeded");
	const failedHistory = useFetchJobHistory("failed");
	const startJobs = useStartFetchJobs();
	const queryClient = useQueryClient();
	const [tab, setTab] = useState<ActivityTab>("all");
	const [search, setSearch] = useState("");
	const [startMessage, setStartMessage] = useState("");
	const activeJobs = activeQuery.data ?? [];
	const activeSourceIds = useMemo(
		() => new Set(activeJobs.map((job) => job.sourceId)),
		[activeJobs],
	);
	const fetchableSources = allSources.filter(
		(source) =>
			source.active &&
			source.type !== "manual" &&
			!activeSourceIds.has(source.id),
	);
	const selectedHistory =
		tab === "succeeded"
			? succeededHistory
			: tab === "failed"
				? failedHistory
				: allHistory;
	const jobs = selectedHistory.data?.pages.flatMap((page) => page.data) ?? [];
	const terminalJobs =
		tab === "all"
			? jobs.filter(
					(job) => job.status === "succeeded" || job.status === "failed",
				)
			: jobs;
	const shownJobs = tab === "in-progress" ? [] : terminalJobs;
	const previousActiveIds = useRef<string>("");

	useEffect(() => {
		if (activeQuery.data === undefined) return;
		const nextIds = activeQuery.data
			.map((job) => job.id)
			.sort()
			.join(",");
		if (previousActiveIds.current && previousActiveIds.current !== nextIds) {
			void queryClient.invalidateQueries({ queryKey: fetchJobKeys.root });
		}
		previousActiveIds.current = nextIds;
	}, [activeQuery.data, queryClient]);

	const onFetchNow = async () => {
		setStartMessage("");
		if (fetchableSources.length === 0) {
			setStartMessage("All active sources already have a run in progress.");
			return;
		}
		const result = await startJobs.mutateAsync(
			fetchableSources.map((source) => source.id),
		);
		setStartMessage(
			result.failed === 0
				? `Started ${result.started} ${result.started === 1 ? "fetch" : "fetches"}.`
				: `Started ${result.started}; ${result.failed} could not be started.`,
		);
	};

	const historyError =
		selectedHistory.error instanceof Error ? selectedHistory.error : null;
	const activeError =
		activeQuery.error instanceof Error ? activeQuery.error : null;
	const historyLoading =
		tab === "in-progress" ? activeQuery.isLoading : selectedHistory.isLoading;
	const displayedHistoryError =
		tab === "in-progress" ? activeError : historyError;
	const retryHistory = () => {
		if (tab === "in-progress") void activeQuery.refetch();
		else void selectedHistory.refetch();
	};

	return (
		<div className="space-y-8">
			<header className="flex justify-end">
				<div className="text-right">
					<Button
						onClick={() => void onFetchNow()}
						disabled={startJobs.isPending}
					>
						<Play data-icon="inline-start" />
						{startJobs.isPending ? "Fetching…" : "Fetch now"}
					</Button>
					{startMessage && (
						<p className="mt-2 text-sm text-muted-foreground" role="status">
							{startMessage}
						</p>
					)}
					{startJobs.isError && (
						<p className="mt-2 text-sm text-destructive" role="alert">
							Could not start fetches. Please try again.
						</p>
					)}
				</div>
			</header>

			<ActiveRunsCard
				jobs={activeJobs}
				sources={allSources}
				isLoading={activeQuery.isLoading}
				error={activeError}
				onRetry={() => void activeQuery.refetch()}
			/>

			<FetchActivityHistory
				jobs={shownJobs}
				activeJobs={activeJobs}
				sources={allSources}
				tab={tab}
				onTabChange={setTab}
				search={search}
				onSearchChange={setSearch}
				isLoading={historyLoading}
				error={displayedHistoryError}
				onRetry={retryHistory}
				hasNextPage={
					tab !== "in-progress" && Boolean(selectedHistory.hasNextPage)
				}
				isFetchingNextPage={selectedHistory.isFetchingNextPage}
				onLoadMore={() => void selectedHistory.fetchNextPage()}
			/>
		</div>
	);
}
