import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Source } from "#/features/feed/types";
import { api, type PageMeta } from "#/lib/api";
import { sourceKeys } from "#/lib/query-keys";

export function useSources(enabled = true) {
	return useQuery({
		queryKey: sourceKeys.all,
		enabled,
		queryFn: async () => {
			const sources: Source[] = [];
			let page = 1;
			let totalPages = 1;
			do {
				const result = await api<Source[], PageMeta>(
					`/sources/?pageSize=100&page=${page}`,
				);
				sources.push(...result.data);
				totalPages = result.meta?.totalPages ?? 1;
				page++;
			} while (page <= totalPages);
			return sources;
		},
	});
}

export function useSidebarSources() {
	return useQuery({
		queryKey: [...sourceKeys.all, "sidebar"],
		queryFn: async () => {
			const { data } = await api<Source[]>(
				"/sources/?activeEqual=true&pageSize=10",
			);
			return data;
		},
	});
}

export function useSaveSource() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async ({
			id,
			includeConfig,
			...values
		}: {
			id?: string;
			includeConfig?: boolean;
			name: string;
			type: string;
			config: Record<string, unknown>;
		}) => {
			if (id) {
				const { data } = await api<Source>(`/sources/${id}`, {
					method: "PATCH",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						name: values.name,
						...(includeConfig ? { config: values.config } : {}),
					}),
				});
				return data;
			}
			const { data } = await api<Source>("/sources/", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					name: values.name,
					type: values.type,
					config: values.config,
				}),
			});
			return data;
		},
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: sourceKeys.all }),
	});
}

export function useSetSourceActive() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
			const { data } = await api<Source>(
				`/sources/${id}/${active ? "reactivate" : "deactivate"}`,
				{ method: "POST" },
			);
			return data;
		},
		onSuccess: (source) => {
			queryClient.setQueryData<Source[]>(sourceKeys.all, (current) =>
				current?.map((item) => (item.id === source.id ? source : item)),
			);
			return queryClient.invalidateQueries({ queryKey: sourceKeys.all });
		},
	});
}

export function usePreviewSource() {
	return useMutation({
		mutationFn: async ({
			signal,
			...values
		}: {
			type: string;
			config: Record<string, unknown>;
			signal: AbortSignal;
		}) =>
			api<{
				entry:
					| import("#/lib/api/schema").components["schemas"]["internal_modules_source_adapter_http.previewEntryResponse"]
					| null;
			}>("/sources/preview", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(values),
				signal,
			}),
	});
}
