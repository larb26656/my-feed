import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ErrorState } from "#/components/ErrorState";
import { Button, buttonVariants } from "#/components/ui/button";
import { useEntries, useToggleEntry } from "#/hooks/queries";
import { FeedFilters } from "./FeedFilters";
import { FeedList } from "./FeedList";
import { FeedSkeleton } from "./FeedSkeleton";
import { useFeedState } from "./FeedState";
import type { FeedFilter } from "./types";
export function FeedPage({
	sourceId,
	savedOnly = false,
}: {
	sourceId?: string;
	savedOnly?: boolean;
}) {
	const [filter, setFilter] = useState<FeedFilter>("ทั้งหมด");
	const { allSources } = useFeedState();
	const [savedFilter, setSavedFilter] = useState(savedOnly);
	const loadMoreRef = useRef<HTMLDivElement>(null);
	const query = useEntries({
		sourceId,
		savedOnly: savedFilter,
		unread: filter === "ยังไม่อ่าน",
	});
	const mutation = useToggleEntry();
	const visible = query.data?.pages.flatMap((page) => page.data) ?? [];
	useEffect(() => {
		const target = loadMoreRef.current;
		if (!target || !query.hasNextPage || query.isFetchingNextPage || query.isFetchNextPageError) {
			return;
		}

		const observer = new IntersectionObserver(
			([entry]) => {
				if (entry.isIntersecting) void query.fetchNextPage();
			},
			{ rootMargin: "200px" },
		);
		observer.observe(target);
		return () => observer.disconnect();
	}, [query.fetchNextPage, query.hasNextPage, query.isFetchingNextPage, query.isFetchNextPageError]);
	return (
		<section>
			{sourceId && (
				<Link
					to="/sources"
					className={buttonVariants({
						variant: "ghost",
						className: "-ml-3 mb-4",
					})}
				>
					<ArrowLeft data-icon="inline-start" />
					Sources ทั้งหมด
				</Link>
			)}
			<div className="mb-6 flex items-center justify-between gap-3">
				<h1 className="text-2xl font-bold">
					{allSources.find((source) => source.id === sourceId)?.name ??
						(savedOnly ? "บันทึกไว้" : "ฟีดของฉัน")}
				</h1>
				<Link
					to="/record-link"
					className={buttonVariants({ variant: "outline", size: "sm" })}
				>
					บันทึกลิงก์
				</Link>
			</div>
			<FeedFilters
				filter={filter}
				setFilter={setFilter}
				savedOnly={savedFilter}
				setSavedOnly={setSavedFilter}
			/>
			{query.isLoading ? (
				<FeedSkeleton />
			) : query.isError ? (
				<ErrorState
					error={query.error}
					message="โหลดรายการไม่สำเร็จ ลองอีกครั้งได้เลย"
					action={
						<Button onClick={() => void query.refetch()}>ลองอีกครั้ง</Button>
					}
				/>
			) : (
				<FeedList
					visible={visible}
					sources={allSources}
					disabled={mutation.isPending}
					onToggle={(entry, action) => mutation.mutate({ entry, action })}
				/>
			)}
			{mutation.isError && <p role="alert">ไม่สามารถอัปเดตรายการได้</p>}
			{query.hasNextPage && (
				<div ref={loadMoreRef} className="flex min-h-12 items-center justify-center">
					{query.isFetchingNextPage && <p role="status">กำลังโหลด...</p>}
					{query.isFetchNextPageError && (
						<div className="flex items-center gap-3">
							<p role="alert">ไม่สามารถโหลดรายการเพิ่มเติมได้</p>
							<Button onClick={() => void query.fetchNextPage()}>ลองอีกครั้ง</Button>
						</div>
					)}
				</div>
			)}
		</section>
	);
}
