export type Entry = {
	id: string;
	sourceId: string;
	title: string;
	link: string;
	description: string;
	thumbnail?: string;
	publishedAt?: string;
	createdAt: string;
	readAt?: string;
	savedAt?: string;
};

export type Source = {
	id: string;
	name: string;
	type: string;
	rssUrl?: string | null;
	active: boolean;
};

export type EntryPage = {
	data: Entry[];
	page: { nextCursor: string | null; hasNext: boolean };
};

export type EntryFilters = {
	sourceId?: string;
	savedOnly: boolean;
	unread: boolean;
};

export type FetchJob = {
	id: string;
	sourceId: string;
	status: "pending" | "running" | "succeeded" | "failed";
	attempt: number;
	maxAttempts: number;
	entriesCreated: number;
};

export const filters = ["ทั้งหมด", "ยังไม่อ่าน"] as const;
export type FeedFilter = (typeof filters)[number];
