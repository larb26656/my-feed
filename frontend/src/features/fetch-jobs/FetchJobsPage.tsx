import { useState } from "react";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { Button } from "#/components/ui/button";
import { Skeleton } from "#/components/ui/skeleton";
import { useFeedState } from "#/features/feed/FeedState";
import type { Source } from "#/features/feed/types";
import { useCreateFetchJob, useFetchJob } from "#/hooks/queries";

const exampleHistory = [
	{
		id: "example-succeeded",
		source: "ตัวอย่าง RSS",
		status: "สำเร็จ",
		entries: 3,
	},
	{ id: "example-failed", source: "ตัวอย่าง RSS", status: "ล้มเหลว", entries: 0 },
];

export function FetchJobsPage() {
	const {
		allSources,
		sourcesLoading,
		sourcesError,
		sourcesErrorValue,
		refetchSources,
	} = useFeedState();
	const fetchableSources = allSources.filter(
		(source: Source) => source.active && source.type !== "manual",
	);
	const [sourceId, setSourceId] = useState("");
	const [jobId, setJobId] = useState<string | null>(null);
	const createJob = useCreateFetchJob();
	const jobQuery = useFetchJob(jobId);
	const job = jobQuery.data;
	const terminal = job?.status === "succeeded" || job?.status === "failed";

	return (
		<div className="space-y-10">
			<header>
				<p className="text-xs font-bold uppercase tracking-widest text-primary">
					เครื่องมือ
				</p>
				<h1 className="mt-2 text-3xl font-bold tracking-tight">Fetch Jobs</h1>
				<p className="mt-2 text-muted-foreground">
					เริ่ม Fetch Source และติดตามสถานะของงานล่าสุด
				</p>
			</header>

			<section
				className="rounded-2xl border bg-card p-6"
				aria-labelledby="start-fetch-title"
			>
				<h2 id="start-fetch-title" className="text-lg font-semibold">
					เริ่ม Fetch
				</h2>
				<div className="mt-4 flex flex-col gap-3 sm:flex-row">
					<label className="flex-1 text-sm font-medium" htmlFor="fetch-source">
						Source ที่ต้องการ Fetch
						<select
							id="fetch-source"
							className="mt-2 block h-11 w-full rounded-lg border bg-background px-3"
							value={sourceId}
							onChange={(event) => setSourceId(event.target.value)}
							disabled={sourcesLoading || fetchableSources.length === 0}
						>
							<option value="">เลือก Source</option>
							{fetchableSources.map((source) => (
								<option key={source.id} value={source.id}>
									{source.name}
								</option>
							))}
						</select>
					</label>
					<button
						type="button"
						className="mt-auto h-11 rounded-lg bg-primary px-5 font-semibold text-primary-foreground disabled:opacity-50"
						disabled={
							!sourceId || createJob.isPending || Boolean(jobId && !terminal)
						}
						onClick={() =>
							createJob.mutate(sourceId, {
								onSuccess: (job) => setJobId(job.id),
							})
						}
					>
						{createJob.isPending ? "กำลังเริ่ม..." : "เริ่ม Fetch"}
					</button>
				</div>
				{sourcesLoading && (
					<output className="mt-3 space-y-2" aria-busy="true">
						<span className="sr-only">กำลังโหลด Sources</span>
						<Skeleton className="h-3 w-28" />
						<Skeleton className="h-11 w-full rounded-lg" />
					</output>
				)}
				{sourcesError && (
					<ErrorState
						size="compact"
						error={sourcesErrorValue ?? "Sources unavailable"}
						message="โหลด Sources ไม่สำเร็จ ลองอีกครั้งได้เลย"
						action={
							<Button onClick={() => void refetchSources()}>ลองอีกครั้ง</Button>
						}
					/>
				)}
				{!sourcesLoading && !sourcesError && fetchableSources.length === 0 && (
					<NoDataState
						size="compact"
						illustration="feed"
						title="ยังไม่มี Source ที่ Fetch ได้"
						description="เพิ่มหรือเปิดใช้งาน RSS Source ก่อนเริ่ม Fetch"
					/>
				)}
				{createJob.isError && (
					<p role="alert" className="mt-3 text-sm text-destructive">
						เริ่ม Fetch ไม่สำเร็จ กรุณาลองอีกครั้ง
					</p>
				)}
			</section>

			<section
				className="rounded-2xl border bg-card p-6"
				aria-labelledby="current-job-title"
			>
				<h2 id="current-job-title" className="text-lg font-semibold">
					Fetch Job ล่าสุด
				</h2>
				{!jobId && (
					<NoDataState
						size="compact"
						illustration="sleeping"
						title="ยังไม่มี Fetch Job"
						description="เลือก Source แล้วเริ่ม Fetch เพื่อดูสถานะงานที่นี่"
					/>
				)}
				{jobQuery.isLoading && (
					<output className="mt-3 space-y-2" aria-busy="true">
						<span className="sr-only">กำลังโหลดสถานะ Fetch Job</span>
						<Skeleton className="h-4 w-48" />
						<Skeleton className="h-4 w-36" />
					</output>
				)}
				{jobQuery.isError && (
					<ErrorState
						size="compact"
						error={jobQuery.error}
						message="โหลดสถานะ Fetch Job ไม่สำเร็จ"
						action={
							<Button onClick={() => void jobQuery.refetch()}>ลองอีกครั้ง</Button>
						}
					/>
				)}
				{job && (
					<div className="mt-3 space-y-1 text-sm">
						<p>
							สถานะ:{" "}
							<strong>
								{job.status === "pending"
									? "รอดำเนินการ"
									: job.status === "running"
										? "กำลังทำงาน"
										: job.status === "succeeded"
											? "สำเร็จ"
											: "ล้มเหลว"}
							</strong>
						</p>
						{job.status === "succeeded" && (
							<p>พบ Entry ใหม่ {job.entriesCreated} รายการ</p>
						)}
						{job.status === "failed" && (
							<p role="alert" className="text-destructive">
								ล้มเหลวหลังลอง {job.attempt} จาก {job.maxAttempts} ครั้ง โดย API
								ไม่ได้ส่งรายละเอียดความผิดพลาดกลับมา
							</p>
						)}
						{!terminal && (
							<p className="text-muted-foreground">กำลังติดตามสถานะ...</p>
						)}
					</div>
				)}
			</section>

			<section
				className="rounded-2xl border bg-card p-6"
				aria-labelledby="example-history-title"
			>
				<h2 id="example-history-title" className="text-lg font-semibold">
					ตัวอย่างประวัติ
				</h2>
				<p className="mt-1 text-sm text-muted-foreground">
					ข้อมูลจำลองเพื่อแสดงตัวอย่างเท่านั้น ไม่ใช่ประวัติ Fetch จริง
				</p>
				<ul className="mt-4 divide-y">
					{exampleHistory.map((item) => (
						<li key={item.id} className="flex justify-between py-3 text-sm">
							<span>{item.source}</span>
							<span>
								{item.status} · {item.entries} Entries
							</span>
						</li>
					))}
				</ul>
			</section>
		</div>
	);
}
