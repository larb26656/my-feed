import {
	Bookmark,
	BookOpenCheck,
	Check,
	CircleCheck,
	Copy,
} from "lucide-react";
import { useState } from "react";
import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import { formatDate } from "#/lib/date";
import { cn } from "#/lib/utils";
import { RichDescription } from "./RichDescription";
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
	const [copied, setCopied] = useState(false);
	const read = !!entry.readAt;
	const saved = !!entry.savedAt;
	const date = entry.publishedAt ?? entry.createdAt;
	const description =
		entry.description?.trim() === entry.title.trim() ? "" : entry.description;
	const copyLink = () => {
		void navigator.clipboard.writeText(entry.link).then(() => {
			setCopied(true);
			window.setTimeout(() => setCopied(false), 2000);
		});
	};

	return (
		<article className="relative flex gap-3 py-5">
			<div
				className="absolute bottom-0 left-5 top-14 w-px bg-border/60"
				aria-hidden="true"
			/>
			<div className="relative z-10 rounded-full ring-4 ring-background">
				<SourceAvatar name={source?.name ?? "Source"} image={source?.image} />
			</div>
			<div className="min-w-0 flex-1">
				<div className="min-h-36">
					<div className="flex items-center justify-between gap-3">
						<div className="flex min-w-0 items-center gap-2">
							<strong className="truncate text-sm">
								{source?.name ?? "Source"}
							</strong>
							{source && <SourceTypeBadge type={source.type} />}
						</div>
						<time
							dateTime={date}
							className="shrink-0 text-xs text-muted-foreground"
						>
							{formatDate(date)}
						</time>
					</div>
					<h3
						className={cn(
							"mt-1 text-sm leading-relaxed",
							read && "text-muted-foreground",
						)}
					>
						<a
							href={entry.link}
							target="_blank"
							rel="noreferrer"
							className="hover:underline"
						>
							{entry.title}
						</a>
					</h3>
					{description && <RichDescription html={description} />}
					{entry.thumbnail && (
						<a
							href={entry.link}
							target="_blank"
							rel="noreferrer"
							className="mt-2 block overflow-hidden rounded-xl"
						>
							<img
								src={entry.thumbnail}
								alt=""
								className="max-h-72 w-full object-cover"
							/>
						</a>
					)}
				</div>
				<div className="mt-2 grid grid-cols-3 text-muted-foreground">
					<button
						type="button"
						disabled={disabled}
						onClick={() => onToggle(entry, "read")}
						aria-label={
							read
								? `Mark ${entry.title} as unread`
								: `Mark ${entry.title} as read`
						}
						aria-pressed={read}
						title={read ? "Read" : "Mark as read"}
						className="inline-flex h-8 items-center justify-center rounded-md hover:bg-muted disabled:opacity-50"
					>
						{read ? <CircleCheck size={19} /> : <BookOpenCheck size={19} />}
					</button>
					<button
						type="button"
						onClick={copyLink}
						aria-label={copied ? "Link copied" : "Copy link"}
						title={copied ? "Link copied" : "Copy link"}
						className="inline-flex h-8 items-center justify-center rounded-md hover:bg-muted"
					>
						{copied ? (
							<Check size={18} strokeWidth={1.9} />
						) : (
							<Copy size={18} strokeWidth={1.7} />
						)}
					</button>
					<button
						type="button"
						disabled={disabled}
						onClick={() => onToggle(entry, "save")}
						aria-label={
							saved ? `Remove ${entry.title} from Saved` : `Save ${entry.title}`
						}
						aria-pressed={saved}
						title={saved ? "Remove from saved" : "Saved"}
						className="inline-flex h-8 items-center justify-center rounded-md hover:bg-muted disabled:opacity-50"
					>
						<Bookmark
							size={18}
							strokeWidth={1.7}
							fill={saved ? "currentColor" : "none"}
						/>
					</button>
				</div>
			</div>
		</article>
	);
}
