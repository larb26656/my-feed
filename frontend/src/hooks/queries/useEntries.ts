import {
	useInfiniteQuery,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import type { Entry, EntryFilters, EntryPage } from "#/features/feed/types";
import { api } from "#/lib/api";
import { entryKeys } from "#/lib/query-keys";

export function useEntries(filters: EntryFilters) {
	return useInfiniteQuery({
		queryKey: entryKeys.list(filters),
		initialPageParam: null as string | null,
		queryFn: async ({ pageParam }): Promise<EntryPage> => {
			const params = new URLSearchParams({
				sortDirection: filters.sortDirection,
			});
			if (filters.sourceId) params.set("sourceId", filters.sourceId);
			if (filters.savedOnly) params.set("savedOnly", "true");
			if (filters.unread) params.set("unread", "true");
			if (pageParam) params.set("cursor", pageParam);
			const result = await api<Entry[], EntryPage["meta"]>(
				`/entries/?${params}`,
			);
			return { data: result.data, meta: result.meta ?? { pageSize: 10 } };
		},
		getNextPageParam: (last) => last.meta.nextCursor || undefined,
	});
}

export function useSavedCount() {
	return useQuery({
		queryKey: entryKeys.savedCount,
		queryFn: async () =>
			(await api<{ count: number }>("/entries/saved-count")).data.count,
	});
}

export function useToggleEntry() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async ({
			entry,
			action,
		}: {
			entry: Entry;
			action: "read" | "save";
		}) => {
			const active = action === "read" ? entry.readAt : entry.savedAt;
			const { data } = await api<Entry>(
				`/entries/${entry.id}/${active ? `clear-${action}` : action}`,
				{ method: "POST" },
			);
			return data;
		},
		onSuccess: async (_updated, { action }) => {
			if (action === "save") {
				await queryClient.invalidateQueries({ queryKey: entryKeys.savedCount });
			}
			await queryClient.invalidateQueries({ queryKey: entryKeys.root });
		},
	});
}

export type SaveLinkInput = {
	title: string;
	link: string;
	description: string;
	thumbnail?: string;
	publishedAt?: string;
};

export function useSaveLink() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async (input: SaveLinkInput) => {
			const { data } = await api<Entry>("/entries/save-link", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(input),
			});
			return data;
		},
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: entryKeys.root }),
	});
}
