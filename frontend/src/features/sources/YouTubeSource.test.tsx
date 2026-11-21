// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { SourcesPage } from "./SourcesPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/sources">{children}</a>
	),
}));
vi.mock("#/features/feed/FeedState", () => ({
	useFeedState: () => ({
		allSources: [],
		sourcesLoading: false,
		sourcesError: false,
	}),
}));
afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("adds a YouTube Source using its channel ID", async () => {
	const fetchMock = vi.fn(async () =>
		okJson({
			id: "yt",
			name: "Channel",
			type: "youtube",
			active: true,
		}),
	);
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourcesPage />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(screen.getAllByRole("button", { name: "Add Source" })[0]);
	await user.click(screen.getByLabelText("YouTube"));
	await user.type(screen.getByLabelText("Source name"), "Channel");
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() => expect(fetchMock).toHaveBeenCalled());
	const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
	expect(JSON.parse(String(init.body))).toEqual({
		name: "Channel",
		type: "youtube",
		youtubeChannelId: "UC1234567890123456789012",
	});
});

test("accepts a canonical channel URL", async () => {
	const fetchMock = vi.fn(async () =>
		okJson({
			id: "yt",
			name: "Channel",
			type: "youtube",
			active: true,
		}),
	);
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider client={new QueryClient()}>
			<SourcesPage />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(screen.getAllByRole("button", { name: "Add Source" })[0]);
	await user.click(screen.getByLabelText("YouTube"));
	await user.type(screen.getByLabelText("Source name"), "Channel");
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"https://www.youtube.com/channel/UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() => expect(fetchMock).toHaveBeenCalled());
});

test("explains supported inputs and rejects a handle", async () => {
	const fetchMock = vi.fn();
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider client={new QueryClient()}>
			<SourcesPage />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(screen.getAllByRole("button", { name: "Add Source" })[0]);
	await user.click(screen.getByLabelText("YouTube"));
	expect(
		screen.getByText(/Handles and custom URLs are not supported/),
	).toBeTruthy();
	await user.type(screen.getByLabelText("Source name"), "Channel");
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"@handle",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(
		await screen.findByText(
			/Enter a channel ID starting with UC or a canonical YouTube channel URL/,
		),
	).toBeTruthy();
	expect(fetchMock).not.toHaveBeenCalled();
});

test("shows duplicate channel conflict", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async () =>
			failJson(409, {
				code: "CONFLICT",
				message: "youtube channel already in use",
			}),
		),
	);
	render(
		<QueryClientProvider client={new QueryClient()}>
			<SourcesPage />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(screen.getAllByRole("button", { name: "Add Source" })[0]);
	await user.click(screen.getByLabelText("YouTube"));
	await user.type(screen.getByLabelText("Source name"), "Channel");
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect((await screen.findByRole("alert")).textContent).toContain(
		"youtube channel already in use",
	);
});
