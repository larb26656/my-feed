import { Clock3, LoaderCircle } from "lucide-react";
import { SourceAvatar } from "#/components/SourceAvatar";
import type { Source } from "#/features/feed/types";
import type { FetchJob } from "../types";
import { FetchJobStatusBadge } from "./FetchJobStatusBadge";

export function ActiveRunRow({
	job,
	source,
}: {
	job: FetchJob;
	source?: Source;
}) {
	const detail =
		job.status === "running"
			? `Attempt ${job.attempt} of ${job.maxAttempts} · Checking for new entries...`
			: job.attempt > 0
				? `Retry queued · Attempt ${job.attempt} of ${job.maxAttempts}`
				: "Waiting for a worker";

	return (
		<article className="flex min-w-0 items-start gap-4 py-4">
			<SourceAvatar
				name={source?.name ?? "Unknown source"}
				imageId={source?.profileImageId}
				sourceId={source?.id}
				size={40}
			/>
			<div className="min-w-0 flex-1">
				<h3 className="truncate font-semibold">
					{source?.name ?? "Unknown source"}
				</h3>
				<p className="truncate text-sm text-muted-foreground">
					{source?.rssUrl ?? "RSS source"}
				</p>
				<p className="mt-1 text-sm text-muted-foreground">{detail}</p>
			</div>
			<div className="flex shrink-0 items-center gap-3 pt-1">
				{job.status === "running" ? (
					<LoaderCircle
						className="size-5 animate-spin text-primary"
						aria-label="Running"
					/>
				) : (
					<Clock3 className="size-5" aria-hidden="true" />
				)}
				<FetchJobStatusBadge job={job} />
			</div>
		</article>
	);
}
