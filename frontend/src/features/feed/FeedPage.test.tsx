// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { FeedPage } from "./FeedPage";
import { FeedProvider, useFeedState } from "./FeedState";

vi.mock("@tanstack/react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/sources">{children}</a>
	),
}));
afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("combines Source, Saved and unread while retaining names of deactivated Sources", async () => {
	const calls: string[] = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			calls.push(url);
			const body = url.includes("/sources/")
				? [
						{ id: "active", name: "Active Source", type: "rss", active: true },
						{
							id: "inactive",
							name: "Deactivated Source",
							type: "rss",
							active: false,
						},
					]
				: {
						data: [
							{
								id: "entry",
								sourceId: "inactive",
								title: "An Entry",
								link: "https://example.com",
								description: "",
								createdAt: "2025-01-01T00:00:00Z",
							},
						],
						page: { nextCursor: null, hasNext: false },
					};
			return { ok: true, json: async () => body };
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<FeedPage sourceId="inactive" />
			</FeedProvider>
		</QueryClientProvider>,
	);
	expect(await screen.findAllByText("Deactivated Source")).toHaveLength(2);
	const user = userEvent.setup();
	await user.click(screen.getByRole("button", { name: "บันทึกไว้" }));
	await user.click(screen.getByRole("button", { name: "ยังไม่อ่าน" }));
	await waitFor(() =>
		expect(
			calls.some((url) => {
				const params = new URL(url, "http://localhost").searchParams;
				return (
					params.get("sourceId") === "inactive" &&
					params.get("savedOnly") === "true" &&
					params.get("unread") === "true"
				);
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
			let body: unknown;
			if (url.includes("/sources/")) {
				body = [
					{ id: "active", name: "Active", type: "rss", active: true },
					{ id: "inactive", name: "Inactive", type: "rss", active: false },
				];
			} else if (url.endsWith("/entries/saved-count")) {
				countRequests++;
				body = { count: Number(saved) };
			} else {
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
				if (init?.method === "POST") body = entry;
				else
					body = { data: [entry], page: { nextCursor: null, hasNext: false } };
			}
			return { ok: true, json: async () => body };
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
		await screen.findByRole("button", { name: "ทำเครื่องหมายว่าอ่านแล้ว" }),
	);
	await screen.findByRole("button", { name: "อ่านแล้ว" });
	await user.click(screen.getByRole("button", { name: "บันทึก An Entry" }));
	await screen.findByText("Navigation: Active / Saved: 1");
	await waitFor(() => expect(countRequests).toBe(2));
	await user.click(
		screen.getByRole("button", { name: "นำ An Entry ออกจาก Saved" }),
	);
	await screen.findByText("Navigation: Active / Saved: 0");
	await waitFor(() => expect(countRequests).toBe(3));
});

test("announces when the persisted Saved count cannot load", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			if (url.includes("/sources/")) return { ok: true, json: async () => [] };
			return { ok: false, json: async () => ({ error: "unavailable" }) };
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
