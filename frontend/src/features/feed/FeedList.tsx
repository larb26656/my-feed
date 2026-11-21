import { Link } from "@tanstack/react-router";
import { NoDataState } from "#/components/NoDataState";
import { buttonVariants } from "#/components/ui/button";
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
			{visible.length === 0 ? (
				<NoDataState
					illustration="feed"
					title="Nothing to read yet"
					description="Add sources to follow or save links you want to read here"
					action={
						<Link to="/record-link" className={buttonVariants()}>
							Save Link
						</Link>
					}
				/>
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
