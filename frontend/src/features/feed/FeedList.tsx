import { FeedItem } from "./FeedItem";
import type { Entry, Source } from "./types";

export function FeedList({
	visible,
	sources,
	disabled,
	onToggle,
}: {
	visible: Entry[];
	sources: Source[];
	disabled: boolean;
	onToggle: (entry: Entry, action: "read" | "save") => void;
}) {
	return (
		<div className="flex flex-col gap-4">
			{visible.map((entry) => (
				<FeedItem
					key={entry.id}
					entry={entry}
					source={sources.find((source) => source.id === entry.sourceId)}
					disabled={disabled}
					onToggle={onToggle}
				/>
			))}
		</div>
	);
}
