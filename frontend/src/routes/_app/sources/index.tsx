import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowRight } from "lucide-react";
import { useFeedState } from "#/features/feed/FeedState";
import { SourceAvatar } from "#/features/feed/SourceAvatar";
export const Route = createFileRoute("/_app/sources/")({ component: Sources });
function Sources() {
	const { sources } = useFeedState();
	return (
		<section>
			<h1 className="text-2xl font-bold">Sources ของคุณ</h1>
			<p className="mt-2 text-sm text-muted-foreground">
				เลือกแหล่งที่มาเพื่อดูบทความ
			</p>
			<div className="mt-8 border-t">
				{sources.map((source) => (
					<Link
						key={source.sourceId}
						to="/sources/$sourceId"
						params={{ sourceId: source.sourceId }}
						className="flex w-full items-center gap-3 border-b py-5 text-left hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring"
					>
						<SourceAvatar entry={source} />
						<span className="min-w-0 flex-1">
							<strong className="block truncate text-sm">
								{source.source}
							</strong>
							<span className="text-xs text-muted-foreground">
								{source.category.toLowerCase()}
							</span>
						</span>
						<ArrowRight className="size-4 text-muted-foreground" />
					</Link>
				))}
			</div>
			<p className="mt-6 text-sm text-muted-foreground">
				การเพิ่มและจัดการ Sources ยังไม่พร้อมใช้งาน
			</p>
		</section>
	);
}
