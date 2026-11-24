import { Skeleton } from "#/components/ui/skeleton";

export function FeedSkeleton() {
	return (
		<output className="flex flex-col gap-4" aria-busy="true">
			<span className="sr-only">Loading entries</span>
			{["first", "second", "third"].map((row) => (
				<article
					key={row}
					className="flex gap-3 bg-card p-4 sm:gap-4 md:rounded-xl md:border md:border-border"
				>
					<Skeleton className="size-8 shrink-0 rounded-full sm:size-10" />
					<div className="min-w-0 flex-1">
						<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-5">
							<div className="mb-1 min-w-0 flex-1">
								<Skeleton className="h-5 w-28" />
								<Skeleton className="mt-1 h-3 w-32" />
								<Skeleton className="mt-2 h-5 w-4/5" />
								<Skeleton className="mt-2 h-4 w-full" />
								<Skeleton className="mt-1 h-4 w-2/3" />
							</div>
							<Skeleton className="aspect-video w-full shrink-0 rounded-xl sm:w-44 lg:w-52" />
						</div>
						<div className="mt-3 flex items-center justify-end gap-2 sm:gap-4">
							{["read", "copy", "save"].map((action) => (
								<div
									key={action}
									className="flex size-11 items-center justify-center sm:size-10"
								>
									<Skeleton className="size-[18px] rounded-md" />
								</div>
							))}
						</div>
					</div>
				</article>
			))}
		</output>
	);
}
