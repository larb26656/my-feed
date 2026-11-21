import type { components } from "./schema";

type Schemas = components["schemas"];

/** Narrowed Entry fields the API always returns in practice. */
export type Entry = {
	id: string;
	sourceId: string;
	title: string;
	link: string;
	description: string;
	createdAt: string;
	thumbnail?: string;
	publishedAt?: string;
	updatedAt?: string;
	readAt?: string;
	savedAt?: string;
};

/** Narrowed Source fields the API always returns in practice. */
export type Source = {
	id: string;
	name: string;
	type: string;
	active: boolean;
	rssUrl?: string | null;
	youtubeChannelId?: string | null;
	createdAt?: string;
	updatedAt?: string;
	/** Client-only avatar override; not part of the wire contract. */
	image?: string | null;
};

export type FetchJobStatus = "pending" | "running" | "succeeded" | "failed";

/** Narrowed Fetch Job fields the API always returns in practice. */
export type FetchJob = {
	id: string;
	sourceId: string;
	status: FetchJobStatus;
	attempt: number;
	maxAttempts: number;
	entriesCreated: number;
	createdAt: string;
	updatedAt: string;
};

export type CursorMeta =
	Schemas["github_com_larb26656_my-feed_backend_internal_platform_http.CursorMeta"];

export type PageMeta =
	Schemas["github_com_larb26656_my-feed_backend_internal_platform_http.PageMeta"];

export type EntryPage = {
	data: Entry[];
	meta: CursorMeta;
};

export type FetchJobsPage = {
	data: FetchJob[];
	meta: CursorMeta;
};

export type SourcesPage = {
	data: Source[];
	meta: PageMeta;
};

export type SaveLinkRequest =
	Schemas["internal_modules_entry_adapter_http.saveLinkRequest"];

export type CreateSourceRequest =
	Schemas["internal_modules_source_adapter_http.createSourceRequest"];

export type UpdateSourceRequest =
	Schemas["internal_modules_source_adapter_http.updateSourceRequest"];
