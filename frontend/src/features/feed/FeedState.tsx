import { createContext, type ReactNode, useContext } from "react";
import { useSavedCount, useSources } from "#/hooks/queries";

function useFeedStateValue() {
	const query = useSources();
	const savedQuery = useSavedCount();
	return {
		savedCount: savedQuery.data,
		savedCountError: savedQuery.isError,
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
