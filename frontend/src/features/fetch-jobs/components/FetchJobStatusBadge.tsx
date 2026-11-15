import { Badge } from "#/components/ui/badge";
import type { FetchJob } from "../types";

export function FetchJobStatusBadge({ job }: { job: FetchJob }) {
	if (job.status === "running") {
		return <Badge>Running</Badge>;
	}
	if (job.status === "pending") {
		return (
			<Badge variant="secondary">
				{job.attempt > 0 ? "Waiting" : "Queued"}
			</Badge>
		);
	}
	if (job.status === "failed") {
		return <Badge variant="destructive">Failed</Badge>;
	}
	return (
		<Badge
			variant="outline"
			className="border-transparent bg-emerald-50 text-emerald-700"
		>
			Succeeded
		</Badge>
	);
}
