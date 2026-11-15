import {
	useInfiniteQuery,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import type {
	FetchJob,
	FetchJobsPage,
	StartFetchJobsResult,
} from "#/features/fetch-jobs/types";
import { api } from "#/lib/api";
import { fetchJobKeys } from "#/lib/query-keys";

const historyPageSize = 20;

export function useFetchJobHistory(status?: "succeeded" | "failed") {
	return useInfiniteQuery({
		queryKey: fetchJobKeys.history(status),
		initialPageParam: null as string | null,
		queryFn: ({ pageParam }) => {
			const params = new URLSearchParams({ pageSize: String(historyPageSize) });
			if (status) params.set("status", status);
			if (pageParam) params.set("cursor", pageParam);
			return api<FetchJobsPage>(`/fetch-jobs/?${params}`);
		},
		getNextPageParam: (lastPage) =>
			lastPage.page.hasNext ? lastPage.page.nextCursor : undefined,
	});
}

export function useActiveFetchJobs() {
	return useQuery({
		queryKey: fetchJobKeys.active,
		queryFn: async () => {
			const [pending, running] = await Promise.all(
				["pending", "running"].map((status) => {
					const params = new URLSearchParams({ status, pageSize: "100" });
					return api<FetchJobsPage>(`/fetch-jobs/?${params}`);
				}),
			);
			return [...pending.data, ...running.data].sort(
				(a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
			);
		},
		refetchInterval: 3000,
	});
}

export function useStartFetchJobs() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async (sourceIds: string[]): Promise<StartFetchJobsResult> => {
			const outcomes = await Promise.allSettled(
				sourceIds.map((sourceId) =>
					api<FetchJob>("/fetch-jobs/", {
						method: "POST",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({ sourceId }),
					}),
				),
			);
			return {
				started: outcomes.filter((outcome) => outcome.status === "fulfilled")
					.length,
				failed: outcomes.filter((outcome) => outcome.status === "rejected")
					.length,
			};
		},
		onSettled: () =>
			queryClient.invalidateQueries({ queryKey: fetchJobKeys.root }),
	});
}

export function useCreateFetchJob() {
	return useMutation({
		mutationFn: (sourceId: string) =>
			api<FetchJob>("/fetch-jobs/", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ sourceId }),
			}),
	});
}

export function useFetchJob(id: string | null) {
	return useQuery({
		queryKey: fetchJobKeys.detail(id),
		queryFn: () => api<FetchJob>(`/fetch-jobs/${id}`),
		enabled: id !== null,
		refetchInterval: (query) => {
			const status = query.state.data?.status;
			return status === "succeeded" || status === "failed" ? false : 1000;
		},
	});
}
