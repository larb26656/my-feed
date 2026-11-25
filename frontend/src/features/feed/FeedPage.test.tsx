// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { FeedPage } from "./FeedPage";
import { FeedProvider, useFeedState } from "./FeedState";
import { SavedPage } from "./SavedPage";
import { SourceFeedPage } from "./SourceFeedPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({
		children,
		to,
		search,
		"aria-label": ariaLabel,
	}: {
		children: React.ReactNode;
		to?: string;
		search?: { sourceId?: string };
		"aria-label"?: string;
	}) => (
		<a
			aria-label={ariaLabel}
			href={search?.sourceId ? `${to}?sourceId=${search.sourceId}` : to}
		>
			{children}
		</a>
	),
}));
afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("shows the feed of a deactivated source", async () => {
	const calls: string[] = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			calls.push(url);
			if (url.endsWith("/entries/saved-count")) {
				return okJson({ count: 0 });
			}
			if (url.includes("/sources/")) {
				return okJson([
					{
						id: "active",
						name: "Active Source",
						type: "rss",
						active: true,
					},
					{
						id: "inactive",
						name: "Deactivated Source",
						type: "rss",
						active: false,
					},
				]);
			}
			return okJson(
				[
					{
						id: "entry",
						sourceId: "inactive",
						title: "An Entry",
						link: "https://example.com",
						description: "",
						createdAt: "2025-01-01T00:00:00Z",
					},
				],
				{ pageSize: 10 },
			);
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourceFeedPage sourceId="inactive" />
			</FeedProvider>
		</QueryClientProvider>,
	);
	expect(await screen.findAllByText("Deactivated Source")).toHaveLength(2);
	expect(
		screen.getByRole("button", { name: "Actions for Deactivated Source" }),
	).toBeTruthy();
	await waitFor(() =>
		expect(
			calls.some((url) => {
				const params = new URL(url, "http://localhost").searchParams;
				return params.get("sourceId") === "inactive";
			}),
		).toBe(true),
	);
});

test("loads the persisted Saved count endpoint and refreshes it after confirmed save and unsave", async () => {
	let saved = false;
	let read = false;
	let countRequests = 0;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (url.includes("/sources/")) {
				return okJson([
					{ id: "active", name: "Active", type: "rss", active: true },
					{ id: "inactive", name: "Inactive", type: "rss", active: false },
				]);
			}
			if (url.endsWith("/entries/saved-count")) {
				countRequests++;
				return okJson({ count: Number(saved) });
			}
			if (init?.method === "POST") {
				if (url.endsWith("/save")) saved = true;
				if (url.endsWith("/clear-save")) saved = false;
				if (url.endsWith("/read")) read = true;
			}
			const entry = {
				id: "entry",
				sourceId: "inactive",
				title: "An Entry",
				link: "https://example.com",
				description: "",
				createdAt: "2025-01-01T00:00:00Z",
				...(saved ? { savedAt: "2025-01-01T00:00:00Z" } : {}),
				...(read ? { readAt: "2025-01-01T00:00:00Z" } : {}),
			};
			if (init?.method === "POST") return okJson(entry);
			return okJson([entry], { pageSize: 10 });
		}),
	);
	function Navigation() {
		const { sources, savedCount } = useFeedState();
		return (
			<div>
				Navigation: {sources.map((source) => source.name).join(", ")} / Saved:{" "}
				{savedCount}
			</div>
		);
	}
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<Navigation />
				<FeedPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	await screen.findByText("Navigation: Active / Saved: 0");
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "Mark An Entry as read" }),
	);
	await screen.findByRole("button", { name: "Mark An Entry as unread" });
	await user.click(screen.getByRole("button", { name: "Save An Entry" }));
	await screen.findByText("Navigation: Active / Saved: 1");
	await waitFor(() => expect(countRequests).toBe(2));
	await user.click(
		screen.getByRole("button", { name: "Remove An Entry from Saved" }),
	);
	await screen.findByText("Navigation: Active / Saved: 0");
	await waitFor(() => expect(countRequests).toBe(3));
});

test("announces when the persisted Saved count cannot load", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			if (url.includes("/sources/")) return okJson([]);
			return failJson(503, { message: "unavailable" });
		}),
	);
	function Navigation() {
		const { savedCountError } = useFeedState();
		return savedCountError ? <p role="alert">Saved count unavailable</p> : null;
	}
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<Navigation />
			</FeedProvider>
		</QueryClientProvider>,
	);
	expect((await screen.findByRole("alert")).textContent).toBe(
		"Saved count unavailable",
	);
});

test("keeps Saved entries in saved-time order from the API", async () => {
	const calls: string[] = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			calls.push(url);
			if (url.includes("/sources/")) {
				return okJson([
					{ id: "news", name: "News", type: "rss", active: true },
				]);
			}
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 2 });
			return okJson(
				[
					{
						id: "saved-later",
						sourceId: "news",
						title: "Saved later",
						link: "https://example.com/later",
						description: "",
						createdAt: "2020-01-01T00:00:00Z",
						publishedAt: "2010-01-01T00:00:00Z",
						savedAt: "2024-06-01T00:00:00Z",
					},
					{
						id: "saved-earlier",
						sourceId: "news",
						title: "Saved earlier",
						link: "https://example.com/earlier",
						description: "",
						createdAt: "2020-01-01T00:00:00Z",
						publishedAt: "2024-01-01T00:00:00Z",
						savedAt: "2024-01-01T00:00:00Z",
					},
				],
				{ pageSize: 10 },
			);
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SavedPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const later = await screen.findByRole("link", { name: "Saved later" });
	const earlier = screen.getByRole("link", { name: "Saved earlier" });
	expect(
		later.compareDocumentPosition(earlier) & Node.DOCUMENT_POSITION_FOLLOWING,
	).toBeTruthy();
	expect(
		calls.some(
			(url) =>
				url.includes("savedOnly=true") && url.includes("savedAtSort=desc"),
		),
	).toBe(true);
	await userEvent
		.setup()
		.click(screen.getByRole("combobox", { name: "Sort entries" }));
	await userEvent
		.setup()
		.click(await screen.findByRole("option", { name: "Oldest first" }));
	await waitFor(() =>
		expect(
			calls.some(
				(url) =>
					url.includes("savedOnly=true") && url.includes("savedAtSort=asc"),
			),
		).toBe(true),
	);
});

test("requests the selected sort direction from the API", async () => {
	const calls: string[] = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			calls.push(url);
			if (url.includes("/sources/")) return okJson([]);
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 0 });
			return okJson([], { pageSize: 10 });
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<FeedPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	await waitFor(() =>
		expect(calls.some((url) => url.includes("publishedAtSort=desc"))).toBe(
			true,
		),
	);
	await userEvent
		.setup()
		.click(screen.getByRole("combobox", { name: "Sort entries" }));
	await userEvent
		.setup()
		.click(await screen.findByRole("option", { name: "Oldest first" }));
	await waitFor(() =>
		expect(calls.some((url) => url.includes("publishedAtSort=asc"))).toBe(true),
	);
	expect(
		screen.queryByRole("combobox", { name: "Sort entries by" }),
	).toBeNull();
});

test.each([
	{ name: "Feed", page: <FeedPage />, hasAdd: true },
	{ name: "Saved", page: <SavedPage />, hasAdd: false },
	{ name: "Source", page: <SourceFeedPage sourceId="news" />, hasAdd: false },
])("shows the appropriate toolbar on $name", async ({ name, page, hasAdd }) => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			if (url.includes("/sources/"))
				return okJson([
					{ id: "news", name: "News", type: "rss", active: true },
				]);
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 0 });
			return okJson([], { pageSize: 10 });
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>{page}</FeedProvider>
		</QueryClientProvider>,
	);
	expect(screen.getByRole("combobox", { name: "Sort entries" })).toBeTruthy();
	expect(Boolean(screen.queryByLabelText("Save Link"))).toBe(hasAdd);
	await screen.findByText(
		name === "Saved" ? "No saved entries yet" : "Nothing to read yet",
	);
	const empty = screen.getByRole("status");
	expect(empty.textContent).toContain(
		name === "Saved" ? "No saved entries yet" : "Nothing to read yet",
	);
	expect(Boolean(empty.querySelector("a"))).toBe(hasAdd);
});

test.each([
	{ name: "Feed toolbar", page: <FeedPage />, target: "Sort entries" },
	{ name: "Saved toolbar", page: <SavedPage />, target: "Sort entries" },
	{
		name: "Source header",
		page: <SourceFeedPage sourceId="news" />,
		target: "All Sources",
	},
])("refreshes from the $name", async ({ page, target }) => {
	let entryRequests = 0;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			if (url.includes("/sources/"))
				return okJson([
					{ id: "news", name: "News", type: "rss", active: true },
				]);
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 0 });
			entryRequests++;
			return okJson([], { pageSize: 10 });
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>{page}</FeedProvider>
		</QueryClientProvider>,
	);
	await screen.findByRole("status");
	await waitFor(() => expect(entryRequests).toBeGreaterThan(0));
	const before = entryRequests;
	const element = screen.queryByLabelText(target) ?? screen.getByText(target);
	fireEvent.touchStart(element, { touches: [{ clientX: 0, clientY: 0 }] });
	fireEvent.touchMove(element, { touches: [{ clientX: 0, clientY: 180 }] });
	fireEvent.touchEnd(element);
	await waitFor(() => expect(entryRequests).toBeGreaterThan(before));
});
