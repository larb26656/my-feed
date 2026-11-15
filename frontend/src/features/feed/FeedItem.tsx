import { Bookmark, Check } from "lucide-react";
import { Button } from "#/components/ui/button";
import { cn } from "#/lib/utils";
import type { Entry, Source } from "./types";
export function FeedItem({
	entry,
	source,
	disabled,
	onToggle,
}: {
	entry: Entry;
	source?: Source;
	disabled: boolean;
	onToggle: (entry: Entry, action: "read" | "save") => void;
}) {
	return (
		<article className="relative flex gap-3 border-b py-6 md:gap-6">
			<div className="min-w-0 flex-1">
				<div className="flex items-center gap-2 text-xs text-muted-foreground">
					<strong className="text-foreground">
						{source?.name ?? "Source"}
					</strong>
					<span>·</span>
					<time dateTime={entry.publishedAt ?? entry.createdAt}>
						{new Date(
							entry.publishedAt ?? entry.createdAt,
						).toLocaleDateString()}
					</time>
				</div>
				<h3
					className={cn(
						"mt-3 text-base font-bold leading-snug md:text-lg",
						entry.readAt && "text-muted-foreground",
					)}
				>
					<a href={entry.link} target="_blank" rel="noreferrer">
						{entry.title}
					</a>
				</h3>
				{entry.description && (
					<p className="mt-2 max-w-xl text-xs leading-relaxed text-muted-foreground">
						{entry.description}
					</p>
				)}
				<Button
					variant="ghost"
					size="xs"
					className="mt-4"
					disabled={disabled}
					onClick={() => onToggle(entry, "read")}
				>
					{entry.readAt ? (
						<>
							<Check data-icon="inline-start" />
							อ่านแล้ว
						</>
					) : (
						"ทำเครื่องหมายว่าอ่านแล้ว"
					)}
				</Button>
			</div>
			{entry.thumbnail && (
				<img
					src={entry.thumbnail}
					alt=""
					className="h-20 w-20 shrink-0 rounded-xl object-cover md:h-24 md:w-28"
				/>
			)}
			<Button
				variant="outline"
				size="icon-sm"
				className="absolute right-0 top-4 rounded-full"
				aria-label={
					entry.savedAt
						? `นำ ${entry.title} ออกจาก Saved`
						: `บันทึก ${entry.title}`
				}
				aria-pressed={!!entry.savedAt}
				disabled={disabled}
				onClick={() => onToggle(entry, "save")}
			>
				<Bookmark fill={entry.savedAt ? "currentColor" : "none"} />
			</Button>
		</article>
	);
}
