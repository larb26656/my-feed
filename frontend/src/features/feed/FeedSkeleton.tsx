import { Skeleton } from "#/components/ui/skeleton";

export function FeedSkeleton() {
	return (
		<output className="mt-2 block" aria-busy="true">
			<span className="sr-only">กำลังโหลดรายการ</span>
			{["first", "second", "third"].map((row) => (
				<article key={row} className="flex gap-3 border-b py-6 md:gap-6">
					<div className="min-w-0 flex-1 space-y-3">
						<div className="flex items-center gap-2">
							<Skeleton className="h-3 w-24" />
							<Skeleton className="h-3 w-20" />
						</div>
						<Skeleton className="h-5 w-4/5 md:h-6" />
						<Skeleton className="h-3 w-full max-w-xl" />
						<Skeleton className="h-3 w-2/3 max-w-xl" />
						<Skeleton className="mt-4 h-7 w-36" />
					</div>
					<Skeleton className="h-20 w-20 shrink-0 rounded-xl md:h-24 md:w-28" />
				</article>
			))}
		</output>
	);
}
