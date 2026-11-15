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
		queryFn: ({ pageParam }) => {
			const params = new URLSearchParams();
			if (filters.sourceId) params.set("sourceId", filters.sourceId);
			if (filters.savedOnly) params.set("savedOnly", "true");
			if (filters.unread) params.set("unread", "true");
			if (pageParam) params.set("cursor", pageParam);
			return api<EntryPage>(`/entries/?${params}`);
		},
		getNextPageParam: (last) =>
			last.page.hasNext ? last.page.nextCursor : undefined,
	});
}

export function useSavedCount() {
	return useQuery({
		queryKey: entryKeys.savedCount,
		queryFn: async () =>
			(await api<{ count: number }>("/entries/saved-count")).count,
	});
}

export function useToggleEntry() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: ({
			entry,
			action,
		}: {
			entry: Entry;
			action: "read" | "save";
		}) => {
			const active = action === "read" ? entry.readAt : entry.savedAt;
			return api<Entry>(
				`/entries/${entry.id}/${active ? `clear-${action}` : action}`,
				{ method: "POST" },
			);
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
		mutationFn: (input: SaveLinkInput) =>
			api<Entry>("/entries/save-link", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(input),
			}),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: entryKeys.root }),
	});
}
