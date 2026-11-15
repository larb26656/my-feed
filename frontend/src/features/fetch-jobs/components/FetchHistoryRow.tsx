import { CheckCircle2, CircleAlert, ChevronRight } from "lucide-react";
import type { Source } from "#/features/feed/types";
import { formatDate } from "#/lib/date";
import type { FetchJob } from "../types";

export function FetchHistoryRow({
	job,
	source,
}: {
	job: FetchJob;
	source?: Source;
}) {
	const failed = job.status === "failed";
	const result = failed
		? `Failed after ${job.attempt} ${job.attempt === 1 ? "attempt" : "attempts"}`
		: job.entriesCreated === 0
			? "No new entries"
			: `${job.entriesCreated} new ${job.entriesCreated === 1 ? "entry" : "entries"}`;

	return (
		<div className="flex min-w-0 items-center gap-4 py-4">
			<span
				className={`flex size-10 shrink-0 items-center justify-center rounded-full ${failed ? "bg-destructive/10 text-destructive" : "bg-emerald-50 text-emerald-700"}`}
			>
				{failed ? (
					<CircleAlert className="size-5" aria-hidden="true" />
				) : (
					<CheckCircle2 className="size-5" aria-hidden="true" />
				)}
			</span>
			<div className="flex min-w-0 flex-1 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
				<div className="min-w-0">
					<p className="truncate font-semibold">
						{source?.name ?? "Unknown source"}
					</p>
					<p className="truncate text-sm text-muted-foreground">
						{source?.rssUrl ?? "RSS source"}
					</p>
				</div>
				<div className="shrink-0 sm:w-44">
					<p
						className={failed ? "font-medium text-destructive" : "font-medium"}
					>
						{result}
					</p>
					<p className="text-sm text-muted-foreground">
						{formatDate(job.createdAt)}
					</p>
				</div>
			</div>
			<ChevronRight
				className="size-5 shrink-0 text-muted-foreground"
				aria-hidden="true"
			/>
		</div>
	);
}
