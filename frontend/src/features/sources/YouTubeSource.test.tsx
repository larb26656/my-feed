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

test("previews an RSS entry without a Source name and clears it when the URL changes", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return okJson({
				entry: {
					title: "First entry",
					link: "https://example.test/first",
					description: "Hello",
				},
			});
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"https://example.test/feed",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	const title = await screen.findByRole("link", { name: "First entry" });
	expect(title.getAttribute("target")).toBe("_blank");
	expect(screen.queryByRole("button", { name: "Copy link" })).toBeNull();
	await user.type(screen.getByLabelText("RSS / Atom feed URL"), "x");
	expect(screen.queryByRole("link", { name: "First entry" })).toBeNull();
});

test("discards a late preview even after the URL changes back", async () => {
	let resolvePreview!: (value: ReturnType<typeof okJson>) => void;
	const fetchMock = vi.fn((url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return new Promise<ReturnType<typeof okJson>>((resolve) => {
				resolvePreview = resolve;
			});
		return Promise.resolve(emptyList(url, init));
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	const input = screen.getByLabelText("RSS / Atom feed URL");
	await user.type(input, "https://example.test/feed");
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	expect(
		(screen.getByRole("button", { name: "Testing..." }) as HTMLButtonElement)
			.disabled,
	).toBe(true);
	await user.type(input, "x");
	await user.keyboard("{Backspace}");
	resolvePreview(
		okJson({ entry: { title: "Stale", link: "https://example.test/stale" } }),
	);
	await waitFor(() => expect(screen.queryByText("Stale")).toBeNull());
});

test("previews a canonical YouTube channel without a Source name and clears on channel edit", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return okJson({
				entry: {
					title: "Video preview",
					link: "https://www.youtube.com/watch?v=first",
					description: "Video description",
					thumbnail: "https://example.test/thumb.jpg",
				},
			});
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.click(screen.getByLabelText("YouTube"));
	const input = screen.getByLabelText("YouTube channel ID or URL");
	await user.type(
		input,
		"https://www.youtube.com/channel/UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	const link = await screen.findByRole("link", { name: "Video preview" });
	expect(link.getAttribute("target")).toBe("_blank");
	expect(screen.queryByRole("button", { name: "Copy link" })).toBeNull();
	const call = fetchMock.mock.calls.find(([url]) =>
		url.endsWith("/sources/preview"),
	);
	expect(JSON.parse(String(call?.[1]?.body))).toMatchObject({
		type: "youtube",
		youtubeChannelId:
			"https://www.youtube.com/channel/UC1234567890123456789012",
	});
	await user.type(input, "x");
	expect(screen.queryByText("Video preview")).toBeNull();
});

test("discards a YouTube preview after switching Source Type away and back", async () => {
	let resolvePreview!: (value: ReturnType<typeof okJson>) => void;
	vi.stubGlobal(
		"fetch",
		vi.fn((url: string, init?: RequestInit) => {
			if (url.endsWith("/sources/preview"))
				return new Promise<ReturnType<typeof okJson>>((resolve) => {
					resolvePreview = resolve;
				});
			return Promise.resolve(emptyList(url, init));
		}),
	);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.click(screen.getByLabelText("YouTube"));
	await user.type(
		screen.getByLabelText("YouTube channel ID or URL"),
		"UC1234567890123456789012",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await user.click(screen.getByLabelText("RSS feed"));
	await user.click(screen.getByLabelText("YouTube"));
	resolvePreview(
		okJson({
			entry: {
				title: "Stale video",
				link: "https://youtube.com/watch?v=stale",
			},
		}),
	);
	await waitFor(() => expect(screen.queryByText("Stale video")).toBeNull());
});

test.each([
	"empty",
	"failure",
])("allows retry and creating a Source after an %s preview", async (outcome) => {
	let attempts = 0;
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview")) {
			attempts++;
			if (attempts === 1)
				return outcome === "empty"
					? okJson({ entry: null })
					: failJson(400, { code: "VALIDATION", message: "Origin failed" });
			return okJson({
				entry: { title: "Retried entry", link: "https://example.test/entry" },
			});
		}
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.type(screen.getByLabelText("Source name"), "News");
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"https://example.test/feed",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByText(
		outcome === "empty" ? "No entries found." : "Origin failed",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByRole("link", { name: "Retried entry" });
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(
				([url, init]) => url.endsWith("/sources/") && init?.method === "POST",
			),
		).toBe(true),
	);
});

test.each([
	"empty",
	"failure",
])("saves directly after an %s preview", async (outcome) => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return outcome === "empty"
				? okJson({ entry: null })
				: failJson(400, { code: "VALIDATION", message: "Origin failed" });
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.type(screen.getByLabelText("Source name"), "News");
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"https://example.test/feed",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByText(
		outcome === "empty" ? "No entries found." : "Origin failed",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(
				([url, init]) => url.endsWith("/sources/") && init?.method === "POST",
			),
		).toBe(true),
	);
});
