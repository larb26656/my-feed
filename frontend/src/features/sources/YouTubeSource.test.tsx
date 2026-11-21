// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { SourcesPage } from "./SourcesPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({
		children,
		...props
	}: ComponentProps<"a"> & { to?: string; params?: unknown }) => (
		<a href="/sources" {...props}>
			{children}
		</a>
	),
}));

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

function isSourcesList(url: string, init?: RequestInit) {
	if (init?.method && init.method !== "GET") return false;
	const path = url.replace(/\?.*$/, "");
	return path === "/api/v1/sources" || path === "/api/v1/sources/";
}

function emptyList(url: string, init?: RequestInit) {
	if (isSourcesList(url, init)) {
		return okJson([], { page: 1, pageSize: 10, total: 0, totalPages: 1 });
	}
	return okJson({
		id: "yt",
		name: "Channel",
		type: "youtube",
		active: true,
	});
}

function renderPage() {
	return render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourcesPage />
		</QueryClientProvider>,
	);
}

test("adds a YouTube Source using its channel ID", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			return okJson({
				id: "yt",
				name: "Channel",
				type: "youtube",
				active: true,
			});
		}
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.click(screen.getByLabelText("YouTube"));
	await user.type(screen.getByLabelText("Source name"), "Channel");
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "POST"),
		).toBe(true),
	);
	const postCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "POST",
	);
	expect(JSON.parse(String(postCall?.[1]?.body))).toEqual({
		name: "Channel",
		type: "youtube",
		youtubeChannelId: "UC1234567890123456789012",
	});
});

test("accepts a canonical channel URL", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			return okJson({
				id: "yt",
				name: "Channel",
				type: "youtube",
				active: true,
			});
		}
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.click(screen.getByLabelText("YouTube"));
	await user.type(screen.getByLabelText("Source name"), "Channel");
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"https://www.youtube.com/channel/UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "POST"),
		).toBe(true),
	);
});

test("explains supported inputs and rejects a handle", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) =>
		emptyList(url, init),
	);
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
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
	expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
		false,
	);
});

test("shows duplicate channel conflict", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string, init?: RequestInit) => {
			if (init?.method === "POST") {
				return failJson(409, {
					code: "CONFLICT",
					message: "youtube channel already in use",
				});
			}
			return emptyList(url, init);
		}),
	);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
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
