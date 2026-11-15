import { Link } from "@tanstack/react-router";
import { buttonVariants } from "#/components/ui/button";
import { NoDataState } from "#/components/NoDataState";
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
		<div>
			{visible.length === 0 ? (
				<NoDataState
					illustration="feed"
					title="ยังไม่มีรายการให้อ่าน"
					description="เพิ่มแหล่งที่มาติดตาม หรือบันทึกลิงก์ที่อยากอ่านไว้ที่นี่"
					action={
						<Link to="/record-link" className={buttonVariants()}>
							บันทึกลิงก์
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
