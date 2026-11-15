import { Skeleton } from "#/components/ui/skeleton";

export function FeedSkeleton() {
	return (
		<output className="mt-2 block" aria-busy="true">
			<span className="sr-only">Loading entries</span>
			{["first", "second", "third"].map((row) => (
				<article key={row} className="relative flex gap-3 py-5">
					<div
						className="absolute bottom-0 left-5 top-14 w-px bg-border/60"
						aria-hidden="true"
					/>
					<Skeleton className="relative z-10 size-10 shrink-0 rounded-full ring-4 ring-background" />
					<div className="min-w-0 flex-1">
						<div className="flex items-center justify-between gap-3">
							<Skeleton className="h-4 w-28" />
							<Skeleton className="h-3 w-16" />
						</div>
						<Skeleton className="mt-2 h-4 w-4/5" />
						<Skeleton className="mt-2 h-3 w-full" />
						<Skeleton className="mt-1 h-3 w-2/3" />
						<Skeleton className="mt-3 h-44 w-full rounded-xl" />
						<div className="mt-2 grid grid-cols-3">
							<Skeleton className="mx-auto h-8 w-8 rounded-md" />
							<Skeleton className="mx-auto h-8 w-8 rounded-md" />
							<Skeleton className="mx-auto h-8 w-8 rounded-md" />
						</div>
					</div>
				</article>
			))}
		</output>
	);
}
