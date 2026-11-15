// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { FeedProvider } from "#/features/feed/FeedState";
import { FetchJobsPage } from "./FetchJobsPage";

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("starts a Fetch Job for a selected fetchable Source and tracks it", async () => {
	const calls: Array<{ url: string; method: string }> = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			const method = init?.method ?? "GET";
			calls.push({ url, method });
			const body = url.endsWith("/sources/?includeDeactivated=true")
				? [
						{
							id: "rss-1",
							name: "Engineering Blog",
							type: "rss",
							active: true,
						},
						{ id: "manual", name: "Saved links", type: "manual", active: true },
						{ id: "inactive", name: "Old Blog", type: "rss", active: false },
					]
				: url.endsWith("/entries/saved-count")
					? { count: 0 }
					: method === "POST"
						? {
								id: "job-1",
								sourceId: "rss-1",
								status: "pending",
								attempt: 0,
								maxAttempts: 3,
								entriesCreated: 0,
							}
						: {
								id: "job-1",
								sourceId: "rss-1",
								status: "succeeded",
								attempt: 1,
								maxAttempts: 3,
								entriesCreated: 4,
							};
			return { ok: true, status: 200, json: async () => body };
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false, refetchInterval: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<FetchJobsPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	const sourceSelect = await screen.findByLabelText("Source to fetch");
	expect(
		await screen.findByRole("option", { name: "Engineering Blog" }),
	).toBeTruthy();
	expect(screen.queryByRole("option", { name: "Saved links" })).toBeNull();
	expect(screen.queryByRole("option", { name: "Old Blog" })).toBeNull();
	await user.selectOptions(sourceSelect, "rss-1");
	await user.click(screen.getByRole("button", { name: "Start Fetch" }));
	await waitFor(() =>
		expect(screen.getByText("Succeeded", { selector: "strong" })).toBeTruthy(),
	);
	expect(screen.getByText(/New entries found: 4/)).toBeTruthy();
	expect(calls).toContainEqual({ url: "/api/v1/fetch-jobs/", method: "POST" });
	expect(calls).toContainEqual({
		url: "/api/v1/fetch-jobs/job-1",
		method: "GET",
	});
});

test("continues polling while a Fetch Job runs and shows a terminal failure", async () => {
	let statusReads = 0;
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (url.endsWith("/sources/?includeDeactivated=true")) {
				return {
					ok: true,
					json: async () => [
						{ id: "rss-1", name: "Blog", type: "rss", active: true },
					],
				};
			}
			if (url.endsWith("/entries/saved-count"))
				return { ok: true, json: async () => ({ count: 0 }) };
			if (init?.method === "POST") {
				return {
					ok: true,
					json: async () => ({
						id: "job-2",
						sourceId: "rss-1",
						status: "pending",
						attempt: 0,
						maxAttempts: 3,
						entriesCreated: 0,
					}),
				};
			}
			statusReads += 1;
			return {
				ok: true,
				json: async () => ({
					id: "job-2",
					sourceId: "rss-1",
					status: statusReads === 1 ? "running" : "failed",
					attempt: 3,
					maxAttempts: 3,
					entriesCreated: 0,
				}),
			};
		}),
	);
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
	const user = userEvent.setup();
	await screen.findByRole("option", { name: "Blog" });
	await user.selectOptions(
		screen.getByLabelText("Source to fetch"),
		"rss-1",
	);
	await user.click(screen.getByRole("button", { name: "Start Fetch" }));
	await screen.findByText("In progress");
	await waitFor(() => expect(statusReads).toBeGreaterThanOrEqual(2), {
		timeout: 3000,
	});
	await waitFor(() =>
		expect(
			screen.getByText(
				"Failed after 3 of 3 attempts. The API did not return error details",
			),
		).toBeTruthy(),
	);
	expect(screen.getByText(/Sample data for display only/)).toBeTruthy();
});

test("shows an error when the backend cannot start a Fetch Job", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) => {
			if (url.endsWith("/sources/?includeDeactivated=true"))
				return {
					ok: true,
					json: async () => [
						{ id: "rss-1", name: "Blog", type: "rss", active: true },
					],
				};
			if (url.endsWith("/entries/saved-count"))
				return { ok: true, json: async () => ({ count: 0 }) };
			return { ok: false, status: 500, json: async () => ({}) };
		}),
	);
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
	const user = userEvent.setup();
	await screen.findByRole("option", { name: "Blog" });
	await user.selectOptions(
		screen.getByLabelText("Source to fetch"),
		"rss-1",
	);
	await user.click(screen.getByRole("button", { name: "Start Fetch" }));
	await screen.findByText("Could not start fetch. Please try again");
});
