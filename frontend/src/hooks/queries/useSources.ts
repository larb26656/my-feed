import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Source } from "#/features/feed/types";
import { api } from "#/lib/api";
import { sourceKeys } from "#/lib/query-keys";

export function useSources() {
	return useQuery({
		queryKey: sourceKeys.all,
		queryFn: () => api<Source[]>("/sources/?includeDeactivated=true"),
	});
}

export function useSaveSource() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			...values
		}: {
			id?: string;
			name: string;
			rssUrl: string;
		}) =>
			id
				? api<Source>(`/sources/${id}`, {
						method: "PATCH",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify(values),
					})
				: api<Source>("/sources/", {
						method: "POST",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({ ...values, type: "rss" }),
					}),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: sourceKeys.all }),
	});
}

export function useSetSourceActive() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({ id, active }: { id: string; active: boolean }) =>
			api<Source>(`/sources/${id}/${active ? "reactivate" : "deactivate"}`, {
				method: "POST",
			}),
		onSuccess: (source) => {
			queryClient.setQueryData<Source[]>(sourceKeys.all, (current) =>
				current?.map((item) => (item.id === source.id ? source : item)),
			);
			return queryClient.invalidateQueries({ queryKey: sourceKeys.all });
		},
	});
}
