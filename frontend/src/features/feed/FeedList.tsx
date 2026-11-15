import { Bookmark } from "lucide-react";
import type { Entry, Source } from "./data";
import { FeedItem } from "./FeedItem";
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
		<div>
			{visible.length === 0 ? (
				<div className="flex flex-col items-center gap-2 py-14 text-center text-muted-foreground">
					<Bookmark />
					<strong className="text-foreground">ยังไม่มีรายการในส่วนนี้</strong>
				</div>
			) : (
				visible.map((entry) => (
					<FeedItem
						key={entry.id}
						entry={entry}
						source={sources.find((source) => source.id === entry.sourceId)}
						disabled={disabled}
						onToggle={onToggle}
					/>
				))
			)}
		</div>
	);
}
