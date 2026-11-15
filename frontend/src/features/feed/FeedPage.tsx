import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { buttonVariants } from "#/components/ui/button";
import { entries, type FeedFilter } from "./data";
import { FeedFilters } from "./FeedFilters";
import { FeedList } from "./FeedList";
import { useFeedState } from "./FeedState";

export function FeedPage({
	sourceId,
	savedOnly = false,
}: {
	sourceId?: string;
	savedOnly?: boolean;
}) {
	const [filter, setFilter] = useState<FeedFilter>("ทั้งหมด");
	const { saved, read, toggleSaved, toggleRead } = useFeedState();
	const source = entries.find((entry) => entry.sourceId === sourceId);
	const visible = entries.filter(
		(entry) =>
			(!savedOnly || saved.includes(entry.id)) &&
			(!sourceId || entry.sourceId === sourceId) &&
			(filter !== "ยังไม่อ่าน" || !read.includes(entry.id)),
	);
	return (
		<section>
			{sourceId && (
				<Link
					to="/sources"
					className={buttonVariants({
						variant: "ghost",
						className: "-ml-3 mb-4",
					})}
				>
					<ArrowLeft data-icon="inline-start" />
					Sources ทั้งหมด
				</Link>
			)}
			<h1 className="mb-6 text-2xl font-bold">
				{source?.source ?? (savedOnly ? "บันทึกไว้" : "ฟีดของฉัน")}
			</h1>
			<FeedFilters filter={filter} setFilter={setFilter} />
			<FeedList
				visible={visible}
				read={read}
				saved={saved}
				onToggleRead={toggleRead}
				onToggleSaved={toggleSaved}
			/>
			<p className="pt-9 text-center text-xs text-muted-foreground">
				✳ &nbsp; อ่านในแบบของคุณ แล้วค่อยกลับมาใหม่ได้เสมอ &nbsp; ✳
			</p>
		</section>
	);
}
