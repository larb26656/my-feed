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

function renderPage() {
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	return render(
		<QueryClientProvider client={client}>
			<SourcesPage />
		</QueryClientProvider>,
	);
}

function isSourcesList(url: string, init?: RequestInit) {
	if (init?.method && init.method !== "GET") return false;
	const path = url.replace(/\?.*$/, "");
	return path === "/api/v1/sources" || path === "/api/v1/sources/";
}

function listMeta(total: number, page = 1, pageSize = 10) {
	return {
		page,
		pageSize,
		total,
		totalPages: Math.max(1, Math.ceil(total / pageSize)),
	};
}

function filterSources(url: string, sources: Array<Record<string, unknown>>) {
	const params = new URL(url, "http://local").searchParams;
	const activeEqual = params.get("activeEqual");
	const nameLike = params.get("nameLike")?.toLowerCase();
	const typeEqual = params.get("typeEqual");
	const page = Number(params.get("page") ?? "1");
	const pageSize = Number(params.get("pageSize") ?? "10");
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

test("creates an RSS Source and shows it in management after success", async () => {
	const sources: Array<Record<string, unknown>> = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (url.includes("/source-types")) return sourceTypesResponse();
			if (init?.method === "POST" && isSourcesList(url, { method: "GET" })) {
				const input = JSON.parse(String(init.body));
				sources.push({ id: "new-source", ...input, active: true });
				return okJson(sources[0], undefined, 201);
			}
			if (isSourcesList(url, init)) {
				const page = filterSources(url, sources);
				return okJson(page.data, page.meta);
			}
			return okJson(null);
		}),
	);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	expect(screen.getByRole("heading", { name: "Add source" })).toBeTruthy();
	await user.type(screen.getByLabelText("Source name"), "Engineering Blog");
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"https://example.com/feed.xml",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await screen.findByRole("link", { name: "Engineering Blog" });
	await waitFor(() => {
		const request = vi
			.mocked(fetch)
			.mock.calls.find(
				([url, init]) =>
					(url === "/api/v1/sources/" || url === "/api/v1/sources") &&
					init?.method === "POST",
			);
		expect(request).toBeDefined();
		expect(JSON.parse(String(request?.[1]?.body))).toEqual({
			name: "Engineering Blog",
			type: "rss",
			config: { url: "https://example.com/feed.xml" },
		});
	});
});

test("edits the supported Source fields and refreshes management", async () => {
	let source = {
		id: "source-1",
		name: "Old name",
		type: "rss",
		config: { url: "https://example.com/old.xml" },
		active: true,
	};
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "PATCH") {
			source = { ...source, ...JSON.parse(String(init.body)) };
			return okJson(source);
		}
		if (isSourcesList(url, init)) {
			const page = filterSources(url, [source]);
			return okJson(page.data, page.meta);
		}
		return okJson(source);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "Actions for Old name" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Edit source" }));
	expect(screen.getByRole("heading", { name: "Edit source" })).toBeTruthy();
	const name = screen.getByLabelText("Source name");
	await user.clear(name);
	await user.type(name, "New name");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await screen.findByRole("button", { name: "Actions for New name" });
	const patchCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "PATCH",
	);
	expect(patchCall?.[0]).toBe("/api/v1/sources/source-1");
	expect(JSON.parse(String(patchCall?.[1]?.body))).toEqual({
		name: "New name",
	});
});

test("shows field validation and server submission errors", async () => {
	const fetchMock = vi.fn(
		async (
			url: string,
			init?: RequestInit,
		): Promise<ReturnType<typeof okJson> | ReturnType<typeof failJson>> => {
			if (isSourcesList(url, init)) return okJson([], listMeta(0));
			return okJson([]);
		},
	);
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(await screen.findByText("Please enter a source name")).toBeTruthy();
	expect(
		await screen.findByText("Please enter a valid RSS / Atom URL"),
	).toBeTruthy();
	expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
		false,
	);
	await user.type(screen.getByLabelText("Source name"), "Good Source");
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"https://example.com/feed.xml",
	);
	fetchMock.mockImplementationOnce(async () => failJson(400));
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(
		await screen.findByText(/Could not save source: Request failed \(400\)/),
	).toBeTruthy();
});

test("moves Sources between Active and Inactive tabs", async () => {
	let sources = [
		{ id: "rss-1", name: "Engineering Blog", type: "rss", active: true },
		{ id: "manual-1", name: "Saved links", type: "manual", active: true },
	];
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			const action = url.endsWith("/deactivate") ? "deactivate" : "reactivate";
			const active = action === "reactivate";
			sources = sources.map((source) =>
				source.id === "rss-1" ? { ...source, active } : source,
			);
			return okJson(sources.find((source) => source.id === "rss-1"));
		}
		if (isSourcesList(url, init)) {
			const page = filterSources(url, sources);
			return okJson(page.data, page.meta);
		}
		return okJson([]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await screen.findByRole("link", { name: "Engineering Blog" });
	expect(screen.getByRole("link", { name: "Saved links" })).toBeTruthy();
	await user.click(
		screen.getByRole("button", { name: "Actions for Engineering Blog" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Deactivate source" }));
	await waitFor(() =>
		expect(screen.queryByRole("link", { name: "Engineering Blog" })).toBeNull(),
	);
	await user.click(screen.getByRole("tab", { name: "Inactive" }));
	expect(
		await screen.findByRole("link", { name: "Engineering Blog" }),
	).toBeTruthy();
	await user.click(
		screen.getByRole("button", { name: "Actions for Engineering Blog" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Activate source" }));
	await waitFor(() =>
		expect(screen.queryByRole("link", { name: "Engineering Blog" })).toBeNull(),
	);
	await user.click(screen.getByRole("tab", { name: "Active" }));
	expect(
		await screen.findByRole("link", { name: "Engineering Blog" }),
	).toBeTruthy();
	expect(
		fetchMock.mock.calls.some(
			([url, init]) =>
				url === "/api/v1/sources/rss-1/deactivate" && init?.method === "POST",
		),
	).toBe(true);
	expect(
		fetchMock.mock.calls.some(
			([url, init]) =>
				url === "/api/v1/sources/rss-1/reactivate" && init?.method === "POST",
		),
	).toBe(true);
	expect(
		screen.queryByRole("button", { name: "Actions for Saved links" }),
	).toBeNull();
});

test("paginates Active Sources with the shared DataTable controls", async () => {
	const sources = Array.from({ length: 12 }, (_, index) => ({
		id: `source-${index + 1}`,
		name: `Source ${index + 1}`,
		type: "rss",
		active: true,
		config: { url: `https://example.com/${index + 1}.xml` },
	}));
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
	const user = userEvent.setup();
	await screen.findByRole("table", { name: "active sources" });
	expect(await screen.findByRole("link", { name: "Source 1" })).toBeTruthy();
	expect(screen.getByRole("link", { name: "Source 10" })).toBeTruthy();
	expect(screen.queryByRole("link", { name: "Source 11" })).toBeNull();
	expect(screen.getByText("Page 1 of 2")).toBeTruthy();
	await user.click(screen.getByRole("button", { name: "Next page" }));
	expect(await screen.findByRole("link", { name: "Source 11" })).toBeTruthy();
	expect(screen.getByRole("link", { name: "Source 12" })).toBeTruthy();
	expect(screen.queryByRole("link", { name: "Source 1" })).toBeNull();
	expect(screen.getByText("Page 2 of 2")).toBeTruthy();
});

test("shows an activation failure without changing a Source", async () => {
	const source = {
		id: "rss-1",
		name: "Engineering Blog",
		type: "rss",
		active: true,
	};
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") return failJson(503);
		if (isSourcesList(url, init)) {
			const page = filterSources(url, [source]);
			return okJson(page.data, page.meta);
		}
		return okJson([source]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "Actions for Engineering Blog" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Deactivate source" }));
	expect(
		await screen.findByText(
			/Could not update source status: Request failed \(503\)/,
		),
	).toBeTruthy();
	expect(
		screen.getByRole("button", { name: "Actions for Engineering Blog" }),
	).toBeTruthy();
});

test("sends search criteria and resets the page", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (isSourcesList(url, init)) return okJson([], listMeta(20));
		return okJson([]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await screen.findByRole("table", { name: "active sources" });
	await user.click(screen.getByRole("button", { name: "Next page" }));
	const callsBeforeEditing = fetchMock.mock.calls.length;
	await user.type(screen.getByLabelText("Search by name"), "blog");
	await user.click(screen.getByRole("combobox", { name: "Type" }));
	await user.click(await screen.findByRole("option", { name: "YouTube" }));
	expect(fetchMock.mock.calls.length).toBe(callsBeforeEditing);
	await user.click(screen.getByRole("button", { name: "Search" }));
	await user.click(screen.getByRole("button", { name: "Date added" }));
	await user.click(screen.getByRole("button", { name: "Date added" }));
	await waitFor(() => {
		const urls = fetchMock.mock.calls.map(
			([url]) => new URL(url, "http://local").searchParams,
		);
		expect(
			urls.some(
				(params) =>
					params.get("nameLike") === "blog" &&
					params.get("typeEqual") === "youtube" &&
					params.get("sortBy") === "createdAt" &&
					params.get("sortDirection") === "desc" &&
					params.get("page") === "1",
			),
		).toBe(true);
	});
	await user.click(screen.getByRole("tab", { name: "Inactive" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([url]) => {
				const params = new URL(url, "http://local").searchParams;
				return (
					params.get("activeEqual") === "false" &&
					params.get("nameLike") === "blog" &&
					params.get("typeEqual") === "youtube"
				);
			}),
		).toBe(true),
	);
});

test("opens management focused on one Source", async () => {
	const sources = [
		{ id: "yt", name: "Fireship", type: "youtube", active: true },
		{ id: "rss", name: "Engineering Blog", type: "rss", active: true },
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
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<SourcesPage focusSourceId="old" />
		</QueryClientProvider>,
	);
	expect(await screen.findByRole("link", { name: "Old Channel" })).toBeTruthy();
	await waitFor(() =>
		expect(screen.queryByRole("link", { name: "Fireship" })).toBeNull(),
	);
	expect(
		(screen.getByLabelText("Search by name") as HTMLInputElement).value,
	).toBe("Old Channel");
	expect(
		screen.getByRole("tab", { name: "Inactive" }).getAttribute("aria-selected"),
	).toBe("true");
});

test("closes the source dialog with Escape and restores focus", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async () => okJson([], listMeta(0))),
	);
	renderPage();
	const user = userEvent.setup();
	const trigger = await screen.findByRole("button", { name: "Add Source" });
	await user.click(trigger);
	expect(screen.getByRole("dialog", { name: "Add source" })).toBeTruthy();
	await user.keyboard("{Escape}");
	await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
	await waitFor(() => expect(document.activeElement).toBe(trigger));
});
