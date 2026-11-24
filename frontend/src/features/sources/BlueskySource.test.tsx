// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { okJson } from "#/lib/api/test-helpers";
import { SourceDialog } from "./SourceDialog";
import { withSourceTypes } from "./sourceTypesTestFixture";

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("offers Bluesky in Add Source and previews an eligible post", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return okJson({
				entry: {
					title: "First line",
					link: "https://bsky.app/profile/did:plc:z72i7hdynmk6r22z27h6tvur/post/3eligible",
					description: "First line\nRest of post",
					publishedAt: "2026-01-02T03:04:05Z",
				},
			});
		if (init?.method === "POST") {
			return okJson({
				id: "bsky",
				name: "Alice",
				type: "bluesky",
				config: { did: "did:plc:z72i7hdynmk6r22z27h6tvur" },
				active: true,
			});
		}
		return okJson([]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourceDialog editing={null} onClose={() => {}} />
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(await screen.findByRole("radio", { name: "Bluesky" }));
	await user.type(
		screen.getByLabelText("Bluesky handle or profile URL"),
		"@alice.bsky.social",
	);
	await user.type(screen.getByLabelText("Source name"), "Alice");
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	const link = await screen.findByRole("link", { name: "First line" });
	expect(link.getAttribute("href")).toBe(
		"https://bsky.app/profile/did:plc:z72i7hdynmk6r22z27h6tvur/post/3eligible",
	);
	const previewCall = fetchMock.mock.calls.find(([url]) =>
		String(url).endsWith("/sources/preview"),
	);
	expect(JSON.parse(String(previewCall?.[1]?.body))).toEqual({
		type: "bluesky",
		config: { did: "@alice.bsky.social" },
	});
});
