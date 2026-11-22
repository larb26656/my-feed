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

type PreviewEntry = {
	title: string;
	link: string;
	description?: string;
	thumbnail?: string;
	publishedAt?: string;
};
type PreviewSource = { name: string; type: string; image?: string | null };
type Props =
	| { preview: true; entry: PreviewEntry; source: PreviewSource }
	| {
			preview?: false;
			entry: Entry;
			source?: Source;
			disabled: boolean;
			onToggle: (entry: Entry, action: "read" | "save") => void;
	  };
export function FeedItem(props: Props) {
	const { entry, source } = props;
	const preview = props.preview === true;

	const [copied, setCopied] = useState(false);
	const [expanded, setExpanded] = useState(false);
	const [overflowing, setOverflowing] = useState(false);
	const read = !preview && !!(entry as Entry).readAt;
	const saved = !preview && !!(entry as Entry).savedAt;
	const date =
		entry.publishedAt ?? (preview ? undefined : (entry as Entry).createdAt);
	const description =
		entry.description?.trim() === entry.title.trim() ? "" : entry.description;
	const copyLink = () => {
		void navigator.clipboard.writeText(entry.link).then(() => {
			setCopied(true);
			window.setTimeout(() => setCopied(false), 2000);
		});
	};

	return (
		<article className="flex gap-3 bg-card p-4 sm:gap-4 md:rounded-xl md:border md:border-border">
			{preview ? (
				source && "image" in source && source.image ? (
					<img
						src={source.image}
						alt={`${source.name} avatar`}
						className="size-8 rounded-full object-cover sm:size-10"
					/>
				) : (
					<SourceAvatar
						name={source?.name ?? "Source"}
						className="!size-8 sm:!size-10"
					/>
				)
			) : (
				<Link
					to="/sources/$sourceId"
					params={{ sourceId: (entry as Entry).sourceId }}
					aria-label={`View ${source?.name ?? "Source"} feed`}
					className="shrink-0 rounded-full focus-visible:outline-2 focus-visible:outline-primary"
				>
					<SourceAvatar
						name={source?.name ?? "Source"}
						imageId={(source as Source | undefined)?.profileImageId}
						sourceId={(source as Source | undefined)?.id}
						className="!size-8 sm:!size-10"
					/>
				</Link>
			)}
			<div className="min-w-0 flex-1">
				<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-5">
					<div className="min-w-0 flex-1 mb-1">
						<div className="min-w-0">
							{preview ? (
								<span className="block truncate text-sm font-semibold">
									{source?.name || "Source"}
								</span>
							) : (
								<Link
									to="/sources/$sourceId"
									params={{ sourceId: (entry as Entry).sourceId }}
									className="block truncate text-sm font-semibold hover:underline"
								>
									{source?.name ?? "Source"}
								</Link>
							)}
							<div className="flex items-center gap-1.5 text-xs text-muted-foreground">
								{source && (
									<SourceTypeBadge type={source.type} variant="inline" />
								)}
								{source && date && <span aria-hidden="true">·</span>}
								{date && <time dateTime={date}>{formatDate(date)}</time>}
							</div>
						</div>
						<h3
							className={cn(
								"mt-2 text-base font-semibold leading-snug",
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
						{description && (
							<RichDescription
								html={description}
								expanded={expanded}
								onOverflowChange={setOverflowing}
							/>
						)}
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
				<div className="mt-3 flex items-center text-muted-foreground">
					{description && overflowing && (
						<button
							type="button"
							aria-expanded={expanded}
							onClick={() => setExpanded(!expanded)}
							className="text-sm font-medium hover:text-foreground hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
						>
							{expanded ? "See less" : "See more"}
						</button>
					)}
					{!preview && (
						<div className="ml-auto flex items-center gap-x-2 sm:gap-x-4">
							<button
								type="button"
								disabled={!preview && props.disabled}
								onClick={() =>
									!preview && props.onToggle(entry as Entry, "read")
								}
								aria-label={
									read
										? `Mark ${entry.title} as unread`
										: `Mark ${entry.title} as read`
								}
								aria-pressed={read}
								title={read ? "Read" : "Mark as read"}
								className="inline-flex min-h-11 min-w-11 items-center justify-center gap-2 sm:min-h-10 sm:min-w-0 rounded-md px-2 text-xs hover:bg-muted disabled:opacity-50"
							>
								{read ? <CircleCheck size={17} /> : <BookOpenCheck size={17} />}
							</button>
							<button
								type="button"
								onClick={copyLink}
								aria-label={copied ? "Link copied" : "Copy link"}
								title={copied ? "Link copied" : "Copy link"}
								className="inline-flex min-h-11 min-w-11 items-center justify-center gap-2 sm:min-h-10 sm:min-w-0 rounded-md px-2 text-xs hover:bg-muted"
							>
								{copied ? (
									<Check size={18} strokeWidth={1.9} />
								) : (
									<Copy size={18} strokeWidth={1.7} />
								)}
							</button>
							<button
								type="button"
								disabled={!preview && props.disabled}
								onClick={() =>
									!preview && props.onToggle(entry as Entry, "save")
								}
								aria-label={
									saved
										? `Remove ${entry.title} from Saved`
										: `Save ${entry.title}`
								}
								aria-pressed={saved}
								title={saved ? "Remove from saved" : "Saved"}
								className="inline-flex min-h-11 min-w-11 items-center justify-center gap-2 sm:min-h-10 sm:min-w-0 rounded-md px-2 text-xs hover:bg-muted disabled:opacity-50"
							>
								<Bookmark
									size={18}
									strokeWidth={1.7}
									fill={saved ? "currentColor" : "none"}
								/>
							</button>
						</div>
					)}
				</div>
			</div>
		</article>
	);
}
