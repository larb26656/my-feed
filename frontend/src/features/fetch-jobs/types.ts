export type { FetchJob, FetchJobsPage } from "#/lib/api";

export type StartFetchJobsResult = {
	started: number;
	failed: number;
};
