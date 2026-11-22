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

const blueskySource = {
	id: "bsky",
	name: "Alice",
	type: "bluesky",
	config: { did: "did:plc:z72i7hdynmk6r22z27h6tvur" },
	active: true,
};

function isSourcesList(url: string, init?: RequestInit) {
	if (init?.method && init.method !== "GET") return false;
	const path = url.replace(/\?.*$/, "");
	return path === "/api/v1/sources" || path === "/api/v1/sources/";
}

test("edits only the name of a Bluesky Source and keeps the profile read-only", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("source-types"))
			return okJson({
				types: [
					{
						code: "bluesky",
						label: "Bluesky",
						schema: {
							type: "object",
							properties: { did: { type: "string", minLength: 1 } },
							required: ["did"],
							additionalProperties: false,
						},
						ui: {
							did: {
								label: "Bluesky handle or profile URL",
								widget: "text",
							},
						},
					},
				],
			});
		if (init?.method === "PATCH") {
			return okJson({ ...blueskySource, name: "Renamed" });
		}
		if (isSourcesList(url, init)) {
			return okJson([blueskySource], {
				page: 1,
				pageSize: 10,
				total: 1,
				totalPages: 1,
			});
		}
		return okJson(blueskySource);
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
		await screen.findByRole("button", { name: "Actions for Alice" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Edit source" }));
	const profile = (await screen.findByLabelText(
		"Bluesky handle or profile URL",
	)) as HTMLInputElement;
	expect(profile.value).toBe("did:plc:z72i7hdynmk6r22z27h6tvur");
	expect(profile.readOnly).toBe(true);
	const name = screen.getByLabelText("Source name");
	await user.clear(name);
	await user.type(name, "Renamed");
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
