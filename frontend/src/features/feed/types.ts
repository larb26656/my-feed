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
	image?: string | null;
	type: string;
	rssUrl?: string | null;
	youtubeChannelId?: string | null;
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

export const filters = ["All", "Unread"] as const;
export type FeedFilter = (typeof filters)[number];
