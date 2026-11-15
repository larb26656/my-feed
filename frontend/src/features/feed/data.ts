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
	active: boolean;
};
export type EntryPage = {
	data: Entry[];
	page: { nextCursor: string | null; hasNext: boolean };
};
export const filters = ["ทั้งหมด", "ยังไม่อ่าน"] as const;
export type FeedFilter = (typeof filters)[number];
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
	const response = await fetch(`/api/v1${path}`, init);
	if (!response.ok) throw new Error(`Request failed (${response.status})`);
	return response.json() as Promise<T>;
}
