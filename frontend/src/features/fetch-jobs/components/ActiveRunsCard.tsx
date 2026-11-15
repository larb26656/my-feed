import { Activity } from "lucide-react";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "#/components/ui/card";
import { Skeleton } from "#/components/ui/skeleton";
import type { Source } from "#/features/feed/types";
import type { FetchJob } from "../types";
import { ActiveRunRow } from "./ActiveRunRow";

export function ActiveRunsCard({
	jobs,
	sources,
	isLoading,
	error,
	onRetry,
}: {
	jobs: FetchJob[];
	sources: Source[];
	isLoading: boolean;
	error?: Error | null;
	onRetry: () => void;
}) {
	const sourceById = new Map(sources.map((source) => [source.id, source]));

	return (
		<Card>
			<CardHeader>
				<div className="flex items-center gap-3">
					<Activity className="size-5 text-primary" aria-hidden="true" />
					<CardTitle>Active runs</CardTitle>
					<Badge variant="secondary">{jobs.length} active</Badge>
				</div>
			</CardHeader>
			<CardContent>
				{error ? (
					<ErrorState
						size="compact"
						error={error}
						message="Could not load active fetches"
						action={<Button onClick={onRetry}>Try again</Button>}
					/>
				) : isLoading ? (
					<div className="flex flex-col gap-4" aria-busy="true">
						<Skeleton className="h-16 w-full" />
						<Skeleton className="h-16 w-full" />
					</div>
				) : jobs.length === 0 ? (
					<NoDataState
						size="compact"
						illustration="sleeping"
						title="Nothing is fetching right now"
						description="New and scheduled runs will show up here."
					/>
				) : (
					<div className="divide-y">
						{jobs.map((job) => (
							<ActiveRunRow
								key={job.id}
								job={job}
								source={sourceById.get(job.sourceId)}
							/>
						))}
					</div>
				)}
			</CardContent>
		</Card>
	);
}
