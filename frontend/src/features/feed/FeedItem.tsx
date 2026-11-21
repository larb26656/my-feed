import { Link } from "@tanstack/react-router";
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
		<article className="flex gap-3 rounded-xl border border-border bg-card p-4 sm:gap-4">
			<Link
				to="/sources/$sourceId"
				params={{ sourceId: entry.sourceId }}
				aria-label={`View ${source?.name ?? "Source"} feed`}
				className="shrink-0 rounded-full focus-visible:outline-2 focus-visible:outline-primary"
			>
				<SourceAvatar name={source?.name ?? "Source"} image={source?.image} />
			</Link>
			<div className="min-w-0 flex-1">
				<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-5">
					<div className="min-w-0 flex-1">
						<div className="flex flex-wrap items-center gap-x-2 gap-y-1">
							<div className="flex min-w-0 items-center gap-2">
								<Link
									to="/sources/$sourceId"
									params={{ sourceId: entry.sourceId }}
									className="truncate text-sm font-semibold hover:underline"
								>
									{source?.name ?? "Source"}
								</Link>
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
								"mt-1 text-base font-semibold leading-snug",
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
					</div>
					{entry.thumbnail && (
						<a
							href={entry.link}
							target="_blank"
							rel="noreferrer"
							className="block w-full shrink-0 overflow-hidden rounded-xl sm:w-44 lg:w-52"
						>
							<img
								src={entry.thumbnail}
								alt=""
								className="aspect-video w-full object-cover"
							/>
						</a>
					)}
				</div>
				<div className="mt-3 flex flex-wrap items-center gap-x-2 text-muted-foreground sm:gap-x-4">
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
						className="inline-flex min-h-10 items-center gap-2 rounded-md px-2 text-xs hover:bg-muted disabled:opacity-50"
					>
						{read ? <CircleCheck size={17} /> : <BookOpenCheck size={17} />}
						<span>{read ? "Read" : "Mark read"}</span>
					</button>
					<button
						type="button"
						onClick={copyLink}
						aria-label={copied ? "Link copied" : "Copy link"}
						title={copied ? "Link copied" : "Copy link"}
						className="inline-flex min-h-10 items-center gap-2 rounded-md px-2 text-xs hover:bg-muted"
					>
						{copied ? (
							<Check size={18} strokeWidth={1.9} />
						) : (
							<Copy size={18} strokeWidth={1.7} />
						)}
						<span>{copied ? "Copied" : "Copy link"}</span>
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
						className="inline-flex min-h-10 items-center gap-2 rounded-md px-2 text-xs hover:bg-muted disabled:opacity-50"
					>
						<Bookmark
							size={18}
							strokeWidth={1.7}
							fill={saved ? "currentColor" : "none"}
						/>
						<span>{saved ? "Saved" : "Save"}</span>
					</button>
				</div>
			</div>
		</article>
	);
}
