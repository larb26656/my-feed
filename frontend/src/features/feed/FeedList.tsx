import { Bookmark } from "lucide-react";
import type { Entry } from "./data";
import { FeedItem } from "./FeedItem";

export function FeedList({
	visible,
	read,
	saved,
	onToggleRead,
	onToggleSaved,
}: {
	visible: Entry[];
	read: number[];
	saved: number[];
	onToggleRead: (id: number) => void;
	onToggleSaved: (id: number) => void;
}) {
	return (
		<div>
			{visible.length === 0 ? (
				<div className="flex flex-col items-center gap-2 py-14 text-center text-muted-foreground">
					<Bookmark />
					<strong className="text-foreground">ยังไม่มีรายการในส่วนนี้</strong>
					<p className="text-xs">ลองเลือกตัวกรองอื่น หรือบันทึกเรื่องที่ชอบไว้ก่อน</p>
				</div>
			) : (
				visible.map((entry) => (
					<FeedItem
						key={entry.id}
						entry={entry}
						isRead={read.includes(entry.id)}
						isSaved={saved.includes(entry.id)}
						onToggleRead={onToggleRead}
						onToggleSaved={onToggleSaved}
					/>
				))
			)}
		</div>
	);
}
