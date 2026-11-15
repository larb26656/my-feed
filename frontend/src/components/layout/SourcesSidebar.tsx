import { ArrowRight } from "lucide-react";
import type { Entry } from "#/features/feed/data";
import { SourceAvatar } from "#/features/feed/SourceAvatar";

export function SourcesSidebar({ entries }: { entries: Entry[] }) {
	return (
		<aside className="hidden border-l bg-card xl:block">
			<div className="p-6">
				<div className="flex justify-between">
					<h3 className="font-bold">Source ของคุณ</h3>
					<span className="text-xs text-muted-foreground">
						{entries.length} sources
					</span>
				</div>
				<p className="mt-2 text-xs text-muted-foreground">
					เรื่องดี ๆ จากแหล่งที่คุณเลือกเอง
				</p>
				<div className="mt-6 flex flex-col gap-5">
					{entries.map((entry) => (
						<div key={entry.id} className="flex items-center gap-3">
							<SourceAvatar entry={entry} />
							<span className="flex min-w-0 flex-1 flex-col">
								<strong className="truncate text-xs">{entry.source}</strong>
								<small className="text-muted-foreground">
									{entry.category.toLowerCase()}
								</small>
							</span>
							<ArrowRight className="size-4 text-muted-foreground" />
						</div>
					))}
				</div>
			</div>
		</aside>
	);
}
