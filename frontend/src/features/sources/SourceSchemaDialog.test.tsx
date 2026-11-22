// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { okJson } from "#/lib/api/test-helpers";
import customSchema from "../../../../testdata/source-custom-schema.json";
import { SourceDialog } from "./SourceDialog";

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});
test("discovers RSS fields and creates a Source with config", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("source-types"))
			return okJson({
				types: [
					{
						code: "rss",
						label: "RSS feed",
						schema: {
							$schema: "https://json-schema.org/draft/2020-12/schema",
							type: "object",
							required: ["url"],
							additionalProperties: false,
							properties: { url: { type: "string", pattern: "^https?://" } },
						},
						ui: { url: { label: "RSS / Atom feed URL", widget: "url" } },
					},
				],
			});
		if (init?.method === "POST")
			return okJson({ id: "new", name: "News", type: "rss", active: true });
		return okJson([]);
	});
	vi.stubGlobal("fetch", fetchMock);
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
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([url]) => url.includes("source-types")),
		).toBe(true),
	);
	await user.type(
		await screen.findByLabelText("RSS / Atom feed URL"),
		"https://example.org/feed",
	);
	await user.type(screen.getByLabelText("Source name"), "News");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "POST"),
		).toBe(true),
	);
	const call = fetchMock.mock.calls.find(([, init]) => init?.method === "POST");
	expect(JSON.parse(String(call?.[1]?.body))).toEqual({
		name: "News",
		type: "rss",
		config: { url: "https://example.org/feed" },
	});
});

test("rejects a URL that does not match the published RSS schema", async () => {
	const fetchMock = vi.fn(async (url: string) =>
		url.includes("source-types")
			? okJson({
					types: [
						{
							code: "rss",
							label: "RSS feed",
							schema: {
								$schema: "https://json-schema.org/draft/2020-12/schema",
								type: "object",
								required: ["url"],
								additionalProperties: false,
								properties: { url: { type: "string", pattern: "^https?://" } },
							},
							ui: { url: { label: "RSS / Atom feed URL", widget: "url" } },
						},
					],
				})
			: okJson({}),
	);
	vi.stubGlobal("fetch", fetchMock);
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
	await waitFor(() => expect(fetchMock).toHaveBeenCalled());
	await user.type(screen.getByLabelText("Source name"), "News");
	await user.type(
		screen.getByLabelText("RSS / Atom feed URL"),
		"ftp://example.org/feed",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(await screen.findByText("Please enter a valid value")).toBeTruthy();
});

test("switching types resets field values and errors and applies defaults", async () => {
	const schema = (key: string, defaultValue?: string) => ({
		$schema: "https://json-schema.org/draft/2020-12/schema",
		type: "object",
		additionalProperties: false,
		required: [key],
		properties: {
			[key]: {
				type: "string",
				pattern: "^ok",
				...(defaultValue ? { default: defaultValue } : {}),
			},
		},
	});
	vi.stubGlobal(
		"fetch",
		vi.fn(async (url: string) =>
			url.includes("source-types")
				? okJson({
						types: [
							{
								code: "first",
								label: "First",
								schema: schema("field"),
								ui: { field: { label: "First field", widget: "text" } },
							},
							{
								code: "second",
								label: "Second",
								schema: schema("other", "ok-default"),
								ui: { other: { label: "Second field", widget: "text" } },
							},
						],
					})
				: okJson({}),
		),
	);
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
	await user.type(await screen.findByLabelText("First field"), "bad");
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	expect(await screen.findByText("Please enter a valid value")).toBeTruthy();
	await user.click(screen.getByLabelText("Second"));
	expect(screen.queryByText("Please enter a valid value")).toBeNull();
	expect(screen.getByLabelText("Second field")).toHaveProperty(
		"value",
		"ok-default",
	);
	await user.click(screen.getByLabelText("First"));
	expect(screen.getByLabelText("First field")).toHaveProperty("value", "");
});

test("edits the stored config using discovered fields", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("source-types"))
			return okJson({
				types: [
					{
						code: "youtube",
						label: "YouTube",
						schema: {
							type: "object",
							required: ["channelId"],
							properties: { channelId: { type: "string", minLength: 1 } },
						},
						ui: { channelId: { label: "Channel ID", widget: "text" } },
					},
				],
			});
		if (init?.method === "PATCH")
			return okJson({
				id: "yt",
				name: "Channel",
				type: "youtube",
				active: true,
			});
		return okJson([]);
	});
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourceDialog
				editing={
					{
						id: "yt",
						name: "Channel",
						type: "youtube",
						config: { channelId: "old" },
						active: true,
					} as never
				}
				onClose={() => {}}
			/>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	const field = await screen.findByLabelText("Channel ID");
	expect((field as HTMLInputElement).value).toBe("old");
	await user.clear(field);
	await user.type(field, "new");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "PATCH"),
		).toBe(true),
	);
	const call = fetchMock.mock.calls.find(
		([, init]) => init?.method === "PATCH",
	);
	expect(JSON.parse(String(call?.[1]?.body))).toEqual({
		name: "Channel",
		config: { channelId: "new" },
	});
});

test("renders registered widgets and submits their typed values", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("source-types"))
			return okJson({
				types: [
					{
						code: "custom",
						label: "Custom",
						schema: {
							...customSchema,
							properties: {
								...customSchema.properties,
								enabled: { type: "boolean" },
							},
						},
						ui: {
							text: { label: "Text", widget: "text" },
							url: { label: "URL", widget: "url" },
							count: { label: "Count", widget: "number" },
							choice: { label: "Choice", widget: "select" },
							enabled: { label: "Enabled", widget: "checkbox" },
						},
					},
				],
			});
		if (init?.method === "POST")
			return okJson({
				id: "new",
				name: "Custom",
				type: "custom",
				active: true,
			});
		return okJson([]);
	});
	vi.stubGlobal("fetch", fetchMock);
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
	await user.type(await screen.findByLabelText("Source name"), "Custom");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	const invalidText = screen.getByLabelText("Text");
	expect(invalidText.getAttribute("aria-required")).toBe("true");
	expect(invalidText.getAttribute("aria-invalid")).toBe("true");
	const errorId = invalidText.getAttribute("aria-describedby");
	expect(errorId && document.getElementById(errorId)?.textContent).toBeTruthy();
	expect(screen.getByLabelText("Choice").getAttribute("aria-invalid")).toBe(
		"true",
	);
	expect(screen.getByLabelText("Enabled").getAttribute("aria-invalid")).toBe(
		"true",
	);
	expect(
		screen.getByLabelText("Enabled").getAttribute("aria-describedby"),
	).toBeTruthy();
	await user.type(screen.getByLabelText("Text"), "ok-value");
	await user.type(screen.getByLabelText("URL"), "https://example.test");
	expect((screen.getByLabelText("Count") as HTMLInputElement).value).toBe("3");
	await user.selectOptions(screen.getByLabelText("Choice"), "2");
	await user.click(screen.getByLabelText("Enabled"));
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "POST"),
		).toBe(true),
	);
	const call = fetchMock.mock.calls.find(([, init]) => init?.method === "POST");
	expect(JSON.parse(String(call?.[1]?.body)).config).toEqual({
		text: "ok-value",
		url: "https://example.test",
		count: 3,
		choice: 2,
		enabled: true,
	});
});

test("shows unsupported widget metadata instead of allowing submission", async () => {
	const fetchMock = vi.fn(async (url: string) =>
		url.includes("source-types")
			? okJson({
					types: [
						{
							code: "custom",
							label: "Custom",
							schema: customSchema,
							ui: { text: { label: "Text", widget: "unknown" } },
						},
					],
				})
			: okJson([]),
	);
	vi.stubGlobal("fetch", fetchMock);
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourceDialog editing={null} onClose={() => {}} />
		</QueryClientProvider>,
	);
	expect(
		await screen.findByText("This Source Type configuration is unsupported"),
	).toBeTruthy();
});
