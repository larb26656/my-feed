export type FetchJob = {
	id: string;
	sourceId: string;
	status: "pending" | "running" | "succeeded" | "failed";
	attempt: number;
	maxAttempts: number;
	entriesCreated: number;
	createdAt: string;
	updatedAt: string;
};

export type FetchJobsPage = {
	data: FetchJob[];
	page: { nextCursor: string | null; hasNext: boolean };
};

export type StartFetchJobsResult = {
	started: number;
	failed: number;
};
