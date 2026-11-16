// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { SourcesPage } from "./SourcesPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/sources">{children}</a>
	),
}));
vi.mock("#/features/feed/FeedState", () => ({
	useFeedState: () => ({
		allSources: [
			{
				id: "yt",
				name: "Channel",
				type: "youtube",
				youtubeChannelId: "UC1234567890123456789012",
				active: true,
			},
		],
		sourcesLoading: false,
		sourcesError: false,
	}),
}));
afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("edits only the name of a YouTube Source", async () => {
	const fetchMock = vi.fn(async () => ({
		ok: true,
		json: async () => ({
			id: "yt",
			name: "Renamed",
			type: "youtube",
			active: true,
		}),
	}));
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider client={new QueryClient()}>
			<SourcesPage />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(screen.getByRole("button", { name: "Actions for Channel" }));
	await user.click(screen.getByRole("menuitem", { name: "Edit source" }));
	const name = screen.getByLabelText("Source name");
	await user.clear(name);
	await user.type(name, "Renamed");
	expect(
		screen.getByText("YouTube channel ID: UC1234567890123456789012"),
	).toBeTruthy();
	expect(screen.queryByLabelText("YouTube channel ID or URL")).toBeNull();
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() => expect(fetchMock).toHaveBeenCalled());
	const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
	expect(JSON.parse(String(init.body))).toEqual({ name: "Renamed" });
});

test("deactivates a YouTube Source through its action menu", async () => {
	const fetchMock = vi.fn(async () => ({
		ok: true,
		json: async () => ({
			id: "yt",
			name: "Channel",
			type: "youtube",
			active: false,
		}),
	}));
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider client={new QueryClient()}>
			<SourcesPage />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(screen.getByRole("button", { name: "Actions for Channel" }));
	await user.click(screen.getByRole("menuitem", { name: "Deactivate source" }));
	await waitFor(() =>
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/sources/yt/deactivate",
			expect.objectContaining({ method: "POST" }),
		),
	);
});
