// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { FeedProvider } from "#/features/feed/FeedState";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { SourcesPage } from "./SourcesPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/sources">{children}</a>
	),
}));

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("creates an RSS Source and shows it in management after success", async () => {
	const sources: Array<Record<string, unknown>> = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (_url: string, init?: RequestInit) => {
			if (init?.method === "POST") {
				const input = JSON.parse(String(init.body));
				sources.push({ id: "new-source", ...input, active: true });
				return okJson(sources[0], undefined, 201);
			}
			return okJson(sources);
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	expect(screen.getByRole("heading", { name: "Add source" })).toBeTruthy();
	await user.type(screen.getByLabelText("Source name"), "Engineering Blog");
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"https://example.com/feed.xml",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await screen.findByText("Engineering Blog");
	await waitFor(() => {
		const request = vi
			.mocked(fetch)
			.mock.calls.find(
				([url, init]) => url === "/api/v1/sources/" && init?.method === "POST",
			);
		expect(request).toBeDefined();
		expect(JSON.parse(String(request?.[1]?.body))).toEqual({
			name: "Engineering Blog",
			type: "rss",
			rssUrl: "https://example.com/feed.xml",
		});
	});
});

test("edits the supported Source fields and refreshes management", async () => {
	let source = {
		id: "source-1",
		name: "Old name",
		type: "rss",
		rssUrl: "https://example.com/old.xml",
		active: true,
	};
	const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
		if (init?.method === "PATCH") {
			source = { ...source, ...JSON.parse(String(init.body)) };
			return okJson(source);
		}
		return okJson([source]);
	});
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
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
		rssUrl: "https://example.com/old.xml",
	});
});

test("shows field validation and server submission errors", async () => {
	const fetchMock = vi.fn(
		async (
			_url: string,
			_init?: RequestInit,
		): Promise<ReturnType<typeof okJson> | ReturnType<typeof failJson>> =>
			okJson([]),
	);
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
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
	fetchMock.mockClear();
	fetchMock.mockImplementationOnce(async () => failJson(400));
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(
		await screen.findByText(/Could not save source: Request failed \(400\)/),
	).toBeTruthy();
});

test("deactivates and reactivates RSS Sources while keeping Entries linked to them", async () => {
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
			return okJson(sources[0]);
		}
		if (url.includes("/entries/")) {
			return okJson([], { pageSize: 10 });
		}
		return okJson(sources);
	});
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "Actions for Engineering Blog" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Deactivate source" }));
	await screen.findByText(/RSS.*Inactive/);
	await user.click(
		screen.getByRole("button", { name: "Actions for Engineering Blog" }),
	);
	expect(screen.getByText(/RSS.*Inactive/)).toBeTruthy();
	await user.click(screen.getByRole("menuitem", { name: "Activate source" }));
	await screen.findByText(/RSS.*Active/);
	expect(screen.getByText(/RSS.*Active/)).toBeTruthy();
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
	expect(screen.queryByRole("button", { name: /Saved links/ })).toBeNull();
});

test("shows an activation failure without changing a Source", async () => {
	const fetchMock = vi.fn(async (_url: string, init?: RequestInit) =>
		init?.method === "POST"
			? failJson(503)
			: okJson([
					{
						id: "rss-1",
						name: "Engineering Blog",
						type: "rss",
						active: true,
					},
				]),
	);
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
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
