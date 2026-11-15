import { Bookmark, Check } from "lucide-react";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { cn } from "#/lib/utils";
import type { Entry } from "./data";
import { artColors, SourceAvatar } from "./SourceAvatar";

export function FeedItem({
	entry,
	isRead,
	isSaved,
	onToggleRead,
	onToggleSaved,
}: {
	entry: Entry;
	isRead: boolean;
	isSaved: boolean;
	onToggleRead: (id: number) => void;
	onToggleSaved: (id: number) => void;
}) {
	return (
		<article className="relative flex gap-3 border-b py-6 md:gap-6">
			<div className="min-w-0 flex-1">
				<div className="flex items-center gap-2 text-xs text-muted-foreground">
					<SourceAvatar entry={entry} />
					<strong className="text-foreground">{entry.source}</strong>
					<span>·</span>
					<span>{entry.time}</span>
				</div>
				<h3
					className={cn(
						"mt-3 text-base font-bold leading-snug md:text-lg",
						isRead && "text-muted-foreground",
					)}
				>
					{entry.title}
				</h3>
				<p className="mt-2 max-w-xl text-xs leading-relaxed text-muted-foreground">
					{entry.description}
				</p>
				<div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
					<Badge variant="secondary">{entry.category}</Badge>
					<span>{entry.read}</span>
					<Button
						variant="ghost"
						size="xs"
						onClick={() => onToggleRead(entry.id)}
					>
						{isRead ? (
							<>
								<Check data-icon="inline-start" />
								อ่านแล้ว
							</>
						) : (
							"ทำเครื่องหมายว่าอ่านแล้ว"
						)}
					</Button>
				</div>
			</div>
			<div
				aria-hidden="true"
				className={cn(
					"flex h-20 w-20 shrink-0 items-end overflow-hidden rounded-xl p-2 text-[10px] font-black leading-tight md:h-24 md:w-28",
					artColors[entry.artwork],
				)}
			>
				<span className="whitespace-pre-line">
					{entry.artwork === "city"
						? "SOMEWHERE\nIN THE CITY"
						: entry.artwork === "creative"
							? "MAKE\nSPACE"
							: entry.artwork === "tech"
								? "A HUMAN\nFUTURE"
								: "SLOW\nSUNDAY"}
				</span>
			</div>
			<Button
				variant="outline"
				size="icon-sm"
				className="absolute right-0 top-4 rounded-full"
				aria-label={
					isSaved ? `นำ ${entry.title} ออกจาก Saved` : `บันทึก ${entry.title}`
				}
				aria-pressed={isSaved}
				onClick={() => onToggleSaved(entry.id)}
			>
				<Bookmark fill={isSaved ? "currentColor" : "none"} />
			</Button>
		</article>
	);
}
