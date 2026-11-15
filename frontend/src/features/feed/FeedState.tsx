import { createContext, useContext, useState, type ReactNode } from "react";
import { entries } from "./data";

const sources = entries.filter(
	(entry, index) =>
		entries.findIndex((item) => item.sourceId === entry.sourceId) === index,
);

function useFeedStateValue() {
	const [saved, setSaved] = useState<number[]>([2]);
	const [read, setRead] = useState<number[]>([]);
	const toggle = (set: typeof setSaved, id: number) =>
		set((current) =>
			current.includes(id)
				? current.filter((item) => item !== id)
				: [...current, id],
		);
	return {
		sources,
		saved,
		read,
		toggleSaved: (id: number) => toggle(setSaved, id),
		toggleRead: (id: number) => toggle(setRead, id),
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
