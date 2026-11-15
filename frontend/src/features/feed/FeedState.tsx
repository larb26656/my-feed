import { useQuery } from "@tanstack/react-query";
import { createContext, type ReactNode, useContext } from "react";
import { api, type EntryPage, type Source } from "./data";

function useFeedStateValue() {
	const query = useQuery({
		queryKey: ["sources"],
		queryFn: () => api<Source[]>("/sources/?includeDeactivated=true"),
	});
	const savedQuery = useQuery({
		queryKey: ["saved-count"],
		queryFn: async ({ signal }) => {
			let count = 0;
			let cursor: string | null = null;
			do {
				const params = new URLSearchParams({
					savedOnly: "true",
					pageSize: "100",
				});
				if (cursor) params.set("cursor", cursor);
				const page = await api<EntryPage>(`/entries/?${params}`, { signal });
				count += page.data.length;
				if (page.page.hasNext && !page.page.nextCursor)
					throw new Error("Missing next cursor");
				cursor = page.page.hasNext ? page.page.nextCursor : null;
			} while (cursor);
			return count;
		},
	});
	return {
		savedCount: savedQuery.data,
		sources: (query.data ?? []).filter((source) => source.active),
		allSources: query.data ?? [],
		sourcesLoading: query.isLoading,
		sourcesError: query.isError,
	};
}
const FeedContext = createContext<ReturnType<typeof useFeedStateValue> | null>(
	null,
);
export function FeedProvider({ children }: { children: ReactNode }) {
	const value = useFeedStateValue();
	return <FeedContext.Provider value={value}>{children}</FeedContext.Provider>;
}
export function useFeedState() {
	const value = useContext(FeedContext);
	if (!value) throw new Error("FeedProvider is missing");
	return value;
}
