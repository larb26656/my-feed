// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { okJson } from "#/lib/api/test-helpers";
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

const youtubeSource = {
	id: "yt",
	name: "Channel",
	type: "youtube",
	config: { channelId: "UC1234567890123456789012" },
	active: true,
};

function isSourcesList(url: string, init?: RequestInit) {
	if (init?.method && init.method !== "GET") return false;
	const path = url.replace(/\?.*$/, "");
	return path === "/api/v1/sources" || path === "/api/v1/sources/";
}

test("edits only the name of a YouTube Source", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("source-types"))
			return okJson({
				types: [
					{
						code: "youtube",
						label: "YouTube",
						schema: {
							type: "object",
							properties: { channelId: { type: "string" } },
						},
						ui: {
							channelId: { label: "YouTube channel ID or URL", widget: "text" },
						},
					},
				],
			});
		if (init?.method === "PATCH") {
			return okJson({ ...youtubeSource, name: "Renamed" });
		}
		if (isSourcesList(url, init)) {
			return okJson([youtubeSource], {
				page: 1,
				pageSize: 10,
				total: 1,
				totalPages: 1,
			});
		}
		return okJson(youtubeSource);
	});
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
	await user.click(
		await screen.findByRole("button", { name: "Actions for Channel" }),
	);
	await user.click(
		await screen.findByRole("menuitem", { name: "Edit source" }),
	);
	await screen.findByLabelText("YouTube channel ID or URL");
	const name = screen.getByLabelText("Source name");
	await user.clear(name);
	await user.type(name, "Renamed");
	expect(
		(screen.getByLabelText("YouTube channel ID or URL") as HTMLInputElement)
			.value,
	).toBe("UC1234567890123456789012");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "PATCH"),
		).toBe(true),
	);
	const patchCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "PATCH",
	);
	expect(JSON.parse(String(patchCall?.[1]?.body))).toEqual({ name: "Renamed" });
});

test("deactivates a YouTube Source through its action menu", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			return okJson({ ...youtubeSource, active: false });
		}
		if (isSourcesList(url, init)) {
			return okJson([youtubeSource], {
				page: 1,
				pageSize: 10,
				total: 1,
				totalPages: 1,
			});
		}
		return okJson(youtubeSource);
	});
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
	await user.click(
		await screen.findByRole("button", { name: "Actions for Channel" }),
	);
	await user.click(
		await screen.findByRole("menuitem", { name: "Deactivate source" }),
	);
	await waitFor(() =>
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/sources/yt/deactivate",
			expect.objectContaining({ method: "POST" }),
		),
	);
});
