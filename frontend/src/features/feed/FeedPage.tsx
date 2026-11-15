import {
	useInfiniteQuery,
	useMutation,
	useQueryClient,
} from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { Button, buttonVariants } from "#/components/ui/button";
import { api, type Entry, type EntryPage, type FeedFilter } from "./data";
import { FeedFilters } from "./FeedFilters";
import { FeedList } from "./FeedList";
import { useFeedState } from "./FeedState";
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
	const client = useQueryClient();
	const query = useInfiniteQuery({
		queryKey: [
			"entries",
			{ sourceId, savedOnly: savedFilter, unread: filter === "ยังไม่อ่าน" },
		],
		initialPageParam: null as string | null,
		queryFn: ({ pageParam }) => {
			const params = new URLSearchParams();
			if (sourceId) params.set("sourceId", sourceId);
			if (savedFilter) params.set("savedOnly", "true");
			if (filter === "ยังไม่อ่าน") params.set("unread", "true");
			if (pageParam) params.set("cursor", pageParam);
			return api<EntryPage>(`/entries/?${params}`);
		},
		getNextPageParam: (last) =>
			last.page.hasNext ? last.page.nextCursor : undefined,
	});
	const mutation = useMutation({
		mutationFn: ({
			entry,
			action,
		}: {
			entry: Entry;
			action: "read" | "save";
		}) => {
			const active = action === "read" ? entry.readAt : entry.savedAt;
			return api<Entry>(
				`/entries/${entry.id}/${active ? `clear-${action}` : action}`,
				{ method: "POST" },
			);
		},
		onSuccess: async (updated, { entry, action }) => {
			if (action === "save") {
				await client.cancelQueries({ queryKey: ["saved-count"] });
				const count = client.getQueryData<number>(["saved-count"]);
				if (count === undefined) {
					void client.invalidateQueries({ queryKey: ["saved-count"] });
				} else {
					client.setQueryData(
						["saved-count"],
						count +
							Number(Boolean(updated.savedAt)) -
							Number(Boolean(entry.savedAt)),
					);
				}
			}
			await client.invalidateQueries({ queryKey: ["entries"] });
		},
	});
	const visible = query.data?.pages.flatMap((page) => page.data) ?? [];
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
			<h1 className="mb-6 text-2xl font-bold">
				{allSources.find((source) => source.id === sourceId)?.name ??
					(savedOnly ? "บันทึกไว้" : "ฟีดของฉัน")}
			</h1>
			<FeedFilters
				filter={filter}
				setFilter={setFilter}
				savedOnly={savedFilter}
				setSavedOnly={setSavedFilter}
			/>
			{query.isLoading ? (
				<p>กำลังโหลด...</p>
			) : query.isError ? (
				<p role="alert">ไม่สามารถโหลดรายการได้</p>
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
				<Button
					onClick={() => void query.fetchNextPage()}
					disabled={query.isFetchingNextPage}
				>
					{query.isFetchingNextPage ? "กำลังโหลด..." : "โหลดเพิ่มเติม"}
				</Button>
			)}
			{query.isFetchNextPageError && (
				<p role="alert">ไม่สามารถโหลดรายการเพิ่มเติมได้</p>
			)}
		</section>
	);
}
