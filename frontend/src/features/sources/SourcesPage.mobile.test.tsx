// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { SourcesPage } from "./SourcesPage";
import { sourceTypesResponse, withSourceTypes } from "./sourceTypesTestFixture";

vi.mock("@tanstack/react-router", () => ({
	Link: ({
		to,
		params,
		children,
		...props
	}: ComponentProps<"a"> & {
		to: string;
		params?: { sourceId?: string };
	}) => {
		const href =
			to === "/sources/$sourceId" && params?.sourceId
				? `/sources/${params.sourceId}`
				: to;
		return (
			<a href={href} {...props}>
				{children}
			</a>
		);
	},
}));

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

function renderPage(props?: { focusSourceId?: string }) {
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	return render(
		<QueryClientProvider client={client}>
			<SourcesPage {...props} />
		</QueryClientProvider>,
	);
}

function stubViewport(matches: boolean) {
	const listeners = new Set<() => void>();
	let current = matches;
	vi.stubGlobal(
		"matchMedia",
		vi.fn((query: string) => ({
			get matches() {
				return current;
			},
			media: query,
			onchange: null,
			addEventListener: (_event: string, listener: () => void) => {
				listeners.add(listener);
			},
			removeEventListener: (_event: string, listener: () => void) => {
				listeners.delete(listener);
			},
			dispatchEvent: () => false,
		})),
	);
	return {
		setMobile(next: boolean) {
			current = next;
			for (const listener of listeners) listener();
		},
	};
}

class ImmediateObserver {
	constructor(private readonly callback: IntersectionObserverCallback) {}
	observe() {
		this.callback(
			[{ isIntersecting: true } as IntersectionObserverEntry],
			this as unknown as IntersectionObserver,
		);
	}
	unobserve() {}
	disconnect() {}
	takeRecords() {
		return [];
	}
	root = null;
	rootMargin = "";
	thresholds = [];
}

function isSourcesList(url: string, init?: RequestInit) {
	if (init?.method && init.method !== "GET") return false;
	const path = url.replace(/\?.*$/, "");
	return path === "/api/v1/sources" || path === "/api/v1/sources/";
}

function listParams(url: string) {
	return new URL(url, "http://local").searchParams;
}

function listMeta(total: number, page = 1, pageSize = 20) {
	return {
		page,
		pageSize,
		total,
		totalPages: Math.max(1, Math.ceil(total / pageSize)),
	};
}

function filterSources(url: string, sources: Array<Record<string, unknown>>) {
	const params = listParams(url);
	const activeEqual = params.get("activeEqual");
	const nameLike = params.get("nameLike")?.toLowerCase();
	const typeEqual = params.get("typeEqual");
	const page = Number(params.get("page") ?? "1");
	const pageSize = Number(params.get("pageSize") ?? "20");
	const filtered = sources.filter((source) => {
		if (activeEqual !== null && String(source.active) !== activeEqual) {
			return false;
		}
		if (
			nameLike &&
			!String(source.name ?? "")
				.toLowerCase()
				.includes(nameLike)
		) {
			return false;
		}
		if (typeEqual && source.type !== typeEqual) return false;
		return true;
	});
	const start = (page - 1) * pageSize;
	return {
		data: filtered.slice(start, start + pageSize),
		meta: listMeta(filtered.length, page, pageSize),
	};
}

function source(index: number, overrides: Record<string, unknown> = {}) {
	return {
		id: `source-${index}`,
		name: `Source ${String(index).padStart(2, "0")}`,
		type: index % 2 === 0 ? "rss" : "youtube",
		active: true,
		config:
			index % 2 === 0
				? { url: `https://example.com/${index}.xml` }
				: { channelId: `UC${String(index).padStart(22, "a")}` },
		...overrides,
	};
}

test("mounts the card list on a phone and the table on a wide screen", async () => {
	const viewport = stubViewport(true);
	const sources = [source(1), source(2)];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (isSourcesList(url, init)) {
				const page = filterSources(url, sources);
				return okJson(page.data, page.meta);
			}
			return okJson([]);
		}),
	);
	renderPage();
	expect(
		await screen.findByRole("list", { name: "active sources" }),
	).toBeTruthy();
	expect(screen.queryByRole("table", { name: "active sources" })).toBeNull();
	expect(screen.getByRole("heading", { name: "Sources" })).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Search" })).toBeNull();
	viewport.setMobile(false);
	expect(
		await screen.findByRole("table", { name: "active sources" }),
	).toBeTruthy();
	expect(screen.queryByRole("list", { name: "active sources" })).toBeNull();
	expect(screen.getByRole("button", { name: "Search" })).toBeTruthy();
});

test("debounces name search and applies the type filter without a Search button", async () => {
	stubViewport(true);
	const sources = [source(1, { name: "Engineering Blog", type: "rss" })];
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (isSourcesList(url, init)) {
			const page = filterSources(url, sources);
			return okJson(page.data, page.meta);
		}
		return okJson([]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await screen.findByRole("list", { name: "active sources" });
	await user.type(screen.getByLabelText("Search sources"), "blog");
	expect(
		fetchMock.mock.calls.some(
			([url]) => listParams(url).get("nameLike") === "blog",
		),
	).toBe(false);
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(
				([url]) => listParams(url).get("nameLike") === "blog",
			),
		).toBe(true),
	);
	await user.click(screen.getByRole("button", { name: "Filter by type" }));
	await user.click(
		await screen.findByRole("menuitemradio", { name: "YouTube" }),
	);
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(
				([url]) =>
					listParams(url).get("typeEqual") === "youtube" &&
					listParams(url).get("nameLike") === "blog" &&
					listParams(url).get("page") === "1" &&
					listParams(url).get("pageSize") === "20",
			),
		).toBe(true),
	);
	expect(screen.queryByRole("button", { name: "Search" })).toBeNull();
});

test("sorts the card list and shows the count with the current sort", async () => {
	stubViewport(true);
	const sources = [source(1), source(2), source(3)];
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (isSourcesList(url, init)) {
			const page = filterSources(url, sources);
			return okJson(page.data, page.meta);
		}
		return okJson([]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	expect(await screen.findByText("3 sources")).toBeTruthy();
	expect(screen.getByText("Name (A-Z)")).toBeTruthy();
	await user.click(screen.getByRole("button", { name: "Sort sources" }));
	await user.click(
		await screen.findByRole("menuitemradio", { name: "Name (Z-A)" }),
	);
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([url]) => {
				const params = listParams(url);
				return params.get("nameSort") === "desc" && params.get("page") === "1";
			}),
		).toBe(true),
	);
	expect(screen.getAllByText("Name (Z-A)").length).toBeGreaterThan(0);
});

test("loads the next page when the list reaches the end", async () => {
	stubViewport(true);
	vi.stubGlobal("IntersectionObserver", ImmediateObserver);
	const sources = Array.from({ length: 25 }, (_, index) => source(index + 1));
	let releaseSecond = () => {};
	const secondPage = new Promise<void>((resolve) => {
		releaseSecond = resolve;
	});
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (isSourcesList(url, init)) {
			if (listParams(url).get("page") === "2") await secondPage;
			const page = filterSources(url, sources);
			return okJson(page.data, page.meta);
		}
		return okJson([]);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	expect(await screen.findByRole("link", { name: "Source 01" })).toBeTruthy();
	expect(screen.queryByRole("link", { name: "Source 21" })).toBeNull();
	expect(await screen.findByText("Loading more")).toBeTruthy();
	expect(screen.getByText("25 sources")).toBeTruthy();
	releaseSecond();
	expect(await screen.findByRole("link", { name: "Source 21" })).toBeTruthy();
	expect(
		fetchMock.mock.calls.some(
			([url]) =>
				listParams(url).get("page") === "2" &&
				listParams(url).get("pageSize") === "20",
		),
	).toBe(true);
	expect(
		fetchMock.mock.calls.some(
			([url]) => listParams(url).get("pageSize") === "10",
		),
	).toBe(false);
});

test("keeps row actions on a card and hides them for the manual Source", async () => {
	stubViewport(true);
	let sources = [
		{ id: "rss-1", name: "Engineering Blog", type: "rss", active: true },
		{ id: "manual-1", name: "Saved links", type: "manual", active: true },
	];
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST" && url.includes("/deactivate")) {
			sources = sources.map((item) =>
				item.id === "rss-1" ? { ...item, active: false } : item,
			);
			return okJson(sources.find((item) => item.id === "rss-1"));
		}
		if (isSourcesList(url, init)) {
			const page = filterSources(url, sources);
			return okJson(page.data, page.meta);
		}
		if (url.includes("/source-types")) return sourceTypesResponse();
		return okJson([]);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	expect(await screen.findByRole("link", { name: "Saved links" })).toBeTruthy();
	expect(
		screen.queryByRole("button", { name: "Actions for Saved links" }),
	).toBeNull();
	await user.click(
		screen.getByRole("button", { name: "Actions for Engineering Blog" }),
	);
	await user.click(
		await screen.findByRole("menuitem", { name: "Deactivate source" }),
	);
	await waitFor(() =>
		expect(screen.queryByRole("link", { name: "Engineering Blog" })).toBeNull(),
	);
	await user.click(screen.getByRole("tab", { name: "Inactive" }));
	expect(
		await screen.findByRole("link", { name: "Engineering Blog" }),
	).toBeTruthy();
});

test("opens management focused on one Source in the card list", async () => {
	stubViewport(true);
	const sources = [
		{ id: "yt", name: "Fireship", type: "youtube", active: true },
		{ id: "old", name: "Old Channel", type: "rss", active: false },
	];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (isSourcesList(url, init)) {
				const page = filterSources(url, sources);
				return okJson(page.data, page.meta);
			}
			return okJson([]);
		}),
	);
	renderPage({ focusSourceId: "old" });
	expect(await screen.findByRole("link", { name: "Old Channel" })).toBeTruthy();
	await waitFor(() =>
		expect(screen.queryByRole("link", { name: "Fireship" })).toBeNull(),
	);
	expect(
		(screen.getByLabelText("Search sources") as HTMLInputElement).value,
	).toBe("Old Channel");
	expect(
		screen.getByRole("tab", { name: "Inactive" }).getAttribute("aria-selected"),
	).toBe("true");
});

test("shows a retry when the card list fails to load", async () => {
	stubViewport(true);
	let fail = true;
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (isSourcesList(url, init)) {
			if (fail) return failJson(503);
			return okJson([source(1, { name: "Recovered" })], listMeta(1));
		}
		return okJson([]);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	expect(
		await screen.findByText("Could not load sources. Please try again"),
	).toBeTruthy();
	fail = false;
	await userEvent
		.setup()
		.click(screen.getByRole("button", { name: "Try again" }));
	expect(await screen.findByRole("link", { name: "Recovered" })).toBeTruthy();
});
