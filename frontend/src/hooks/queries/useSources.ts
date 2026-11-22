import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Source } from "#/features/feed/types";
import { api, type PageMeta, type SourcesPage } from "#/lib/api";
import { sourceKeys } from "#/lib/query-keys";

export type SourcesPageSize = 10 | 20 | 50;
export type SourceFilters = {
	name: string;
	type: string;
};

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

export function useSourcesPage(
	active: boolean,
	page: number,
	pageSize: SourcesPageSize,
	filters: SourceFilters,
	sort: { sortBy: string; sortDirection: string },
) {
	return useQuery({
		queryKey: sourceKeys.page(active, page, pageSize, filters, sort),
		queryFn: async (): Promise<SourcesPage> => {
			const params = new URLSearchParams({
				activeEqual: String(active),
				page: String(page),
				pageSize: String(pageSize),
				sortBy: sort.sortBy,
				sortDirection: sort.sortDirection,
			});
			if (filters.name.trim()) params.set("nameLike", filters.name.trim());
			if (filters.type) params.set("typeEqual", filters.type);
			const result = await api<Source[], PageMeta>(`/sources/?${params}`);
			return {
				data: result.data,
				meta: result.meta ?? {
					page,
					pageSize,
					total: result.data.length,
					totalPages: 1,
				},
			};
		},
		placeholderData: (previous) => previous,
	});
}

export function useSaveSource() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async ({
			id,
			...values
		}: {
			id?: string;
			name: string;
			rssUrl: string;
			type: "rss" | "youtube";
			youtubeChannelId: string;
		}) => {
			if (id) {
				const { data } = await api<Source>(`/sources/${id}`, {
					method: "PATCH",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify(
						values.type === "youtube"
							? { name: values.name }
							: { name: values.name, rssUrl: values.rssUrl },
					),
				});
				return data;
			}
			const { data } = await api<Source>("/sources/", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(
					values.type === "youtube"
						? {
								name: values.name,
								type: values.type,
								youtubeChannelId: values.youtubeChannelId.trim(),
							}
						: {
								name: values.name,
								type: values.type,
								rssUrl: values.rssUrl,
							},
				),
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
