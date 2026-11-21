// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { FeedProvider } from "#/features/feed/FeedState";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { FetchJobsPage } from "./FetchJobsPage";

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

const sources = [
	{ id: "rss-1", name: "Engineering Blog", type: "rss", active: true },
	{ id: "manual", name: "Saved links", type: "manual", active: true },
	{ id: "inactive", name: "Old Blog", type: "rss", active: false },
];
const page = (data: unknown[] = []) => okJson(data, { pageSize: 20 });
const job = {
	id: "job-1",
	sourceId: "rss-1",
	status: "succeeded",
	attempt: 1,
	maxAttempts: 3,
	entriesCreated: 4,
	createdAt: "2025-01-01T00:00:00Z",
	updatedAt: "2025-01-01T00:00:00Z",
};

function renderPage() {
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<FetchJobsPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
}

test("starts fetches only for available Sources and displays completed history", async () => {
	const requests: string[] = [];
	let started = false;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (url.includes("/sources/")) return okJson(sources);
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 0 });
			if (init?.method === "POST") {
				requests.push(init.body as string);
				started = true;
				return okJson(job, undefined, 201);
			}
			const status = new URL(url, "http://localhost").searchParams.get(
				"status",
			);
			return page(started && (!status || status === "succeeded") ? [job] : []);
		}),
	);
	renderPage();
	await screen.findByText("Nothing is fetching right now");
	await userEvent
		.setup()
		.click(screen.getByRole("button", { name: "Fetch now" }));
	await screen.findByText("Started 1 fetch.");
	await screen.findByText("4 new entries");
	expect(requests).toEqual([JSON.stringify({ sourceId: "rss-1" })]);
});

test("shows running and failed jobs from the API", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			if (url.includes("/sources/")) return okJson(sources);
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 0 });
			const status = new URL(url, "http://localhost").searchParams.get(
				"status",
			);
			return page(
				status === "running"
					? [{ ...job, status: "running" }]
					: !status || status === "failed"
						? [{ ...job, status: "failed", attempt: 3 }]
						: [],
			);
		}),
	);
	renderPage();
	await screen.findByText(/Attempt 1 of 3/);
	await screen.findByText("Failed after 3 attempts");
});

test("reports when a fetch cannot be started", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (init?.method === "POST") return failJson(500);
			if (url.includes("/sources/")) return okJson(sources);
			if (url.endsWith("/entries/saved-count")) return okJson({ count: 0 });
			return page();
		}),
	);
	renderPage();
	await screen.findByText("Nothing is fetching right now");
	await userEvent
		.setup()
		.click(screen.getByRole("button", { name: "Fetch now" }));
	expect(
		await screen.findByText("Started 0; 1 could not be started."),
	).toBeTruthy();
});
