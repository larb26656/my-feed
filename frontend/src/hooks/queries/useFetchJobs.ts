import { useMutation, useQuery } from "@tanstack/react-query";
import type { FetchJob } from "#/features/feed/types";
import { api } from "#/lib/api";
import { fetchJobKeys } from "#/lib/query-keys";

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
