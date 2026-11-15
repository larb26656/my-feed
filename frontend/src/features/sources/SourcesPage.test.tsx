// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { FeedProvider } from "#/features/feed/FeedState";
import { SourcesPage } from "./SourcesPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({ children }: { children: React.ReactNode }) => (
		<a href="/sources">{children}</a>
	),
}));

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

test("creates an RSS Source and shows it in management after success", async () => {
	const sources: Array<Record<string, unknown>> = [];
	vi.stubGlobal(
		"fetch",
		vi.fn(async (_url: string, init?: RequestInit) => {
			if (init?.method === "POST") {
				const input = JSON.parse(String(init.body));
				sources.push({ id: "new-source", ...input, active: true });
			}
			return {
				ok: true,
				status: init?.method === "POST" ? 201 : 200,
				json: async () => (init?.method === "POST" ? sources[0] : sources),
			};
		}),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "เพิ่ม Source" }));
	await user.type(screen.getByLabelText("ชื่อ Source"), "Engineering Blog");
	await user.type(
		screen.getByLabelText("RSS URL"),
		"https://example.com/feed.xml",
	);
	await user.click(screen.getByRole("button", { name: "บันทึก Source" }));
	await screen.findByText("Engineering Blog");
	await waitFor(() => {
		const request = vi
			.mocked(fetch)
			.mock.calls.find(
				([url, init]) => url === "/api/v1/sources/" && init?.method === "POST",
			);
		expect(request).toBeDefined();
		expect(JSON.parse(String(request?.[1]?.body))).toEqual({
			name: "Engineering Blog",
			type: "rss",
			rssUrl: "https://example.com/feed.xml",
		});
	});
});

test("edits the supported Source fields and refreshes management", async () => {
	let source = {
		id: "source-1",
		name: "Old name",
		type: "rss",
		rssUrl: "https://example.com/old.xml",
		active: true,
	};
	const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
		if (init?.method === "PATCH") {
			source = { ...source, ...JSON.parse(String(init.body)) };
		}
		return {
			ok: true,
			json: async () => (init?.method === "PATCH" ? source : [source]),
		};
	});
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "แก้ไข Old name" }),
	);
	const name = screen.getByLabelText("ชื่อ Source");
	await user.clear(name);
	await user.type(name, "New name");
	await user.click(screen.getByRole("button", { name: "บันทึก Source" }));
	await screen.findByRole("button", { name: "แก้ไข New name" });
	const patchCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "PATCH",
	);
	expect(patchCall?.[0]).toBe("/api/v1/sources/source-1");
	expect(JSON.parse(String(patchCall?.[1]?.body))).toEqual({
		name: "New name",
		rssUrl: "https://example.com/old.xml",
	});
});

test("shows field validation and server submission errors", async () => {
	const fetchMock = vi.fn(
		async (
			_url: string,
			_init?: RequestInit,
		): Promise<{
			ok: boolean;
			status?: number;
			json: () => Promise<unknown>;
		}> => ({
			ok: true,
			json: async () => [],
		}),
	);
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(await screen.findByRole("button", { name: "เพิ่ม Source" }));
	await user.click(screen.getByRole("button", { name: "บันทึก Source" }));
	expect(await screen.findByText("กรุณาระบุชื่อ Source")).toBeTruthy();
	expect(await screen.findByText("กรุณาระบุ RSS URL ที่ถูกต้อง")).toBeTruthy();
	expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
		false,
	);
	await user.type(screen.getByLabelText("ชื่อ Source"), "Good Source");
	await user.type(
		screen.getByLabelText("RSS URL"),
		"https://example.com/feed.xml",
	);
	fetchMock.mockClear();
	fetchMock.mockImplementationOnce(async () => ({
		ok: false,
		status: 400,
		json: async () => ({}),
	}));
	await user.click(screen.getByRole("button", { name: "บันทึก Source" }));
	expect(
		await screen.findByText(/บันทึก Source ไม่สำเร็จ: Request failed \(400\)/),
	).toBeTruthy();
});

test("deactivates and reactivates RSS Sources while keeping Entries linked to them", async () => {
	let sources = [
		{ id: "rss-1", name: "Engineering Blog", type: "rss", active: true },
		{ id: "manual-1", name: "Saved links", type: "manual", active: true },
	];
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			const action = url.endsWith("/deactivate") ? "deactivate" : "reactivate";
			const active = action === "reactivate";
			sources = sources.map((source) =>
				source.id === "rss-1" ? { ...source, active } : source,
			);
			return { ok: true, status: 200, json: async () => sources[0] };
		}
		if (url.includes("/entries/")) {
			return {
				ok: true,
				json: async () => ({
					data: [],
					page: { nextCursor: null, hasNext: false },
				}),
			};
		}
		return { ok: true, json: async () => sources };
	});
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "ปิดใช้งาน Engineering Blog" }),
	);
	await screen.findByRole("button", { name: "เปิดใช้งาน Engineering Blog" });
	expect(screen.getByText(/RSS.*ปิดใช้งาน/)).toBeTruthy();
	await user.click(
		screen.getByRole("button", { name: "เปิดใช้งาน Engineering Blog" }),
	);
	await screen.findByRole("button", { name: "ปิดใช้งาน Engineering Blog" });
	expect(screen.getByText(/RSS.*ใช้งานอยู่/)).toBeTruthy();
	expect(
		fetchMock.mock.calls.some(
			([url, init]) =>
				url === "/api/v1/sources/rss-1/deactivate" && init?.method === "POST",
		),
	).toBe(true);
	expect(
		fetchMock.mock.calls.some(
			([url, init]) =>
				url === "/api/v1/sources/rss-1/reactivate" && init?.method === "POST",
		),
	).toBe(true);
	expect(screen.queryByRole("button", { name: /Saved links/ })).toBeNull();
});

test("shows an activation failure without changing a Source", async () => {
	const fetchMock = vi.fn(async (_url: string, init?: RequestInit) =>
		init?.method === "POST"
			? { ok: false, status: 503, json: async () => ({}) }
			: {
					ok: true,
					json: async () => [
						{
							id: "rss-1",
							name: "Engineering Blog",
							type: "rss",
							active: true,
						},
					],
				},
	);
	vi.stubGlobal("fetch", fetchMock);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	render(
		<QueryClientProvider client={client}>
			<FeedProvider>
				<SourcesPage />
			</FeedProvider>
		</QueryClientProvider>,
	);
	await userEvent
		.setup()
		.click(
			await screen.findByRole("button", { name: "ปิดใช้งาน Engineering Blog" }),
		);
	expect(
		await screen.findByText(
			/เปลี่ยนสถานะ Source ไม่สำเร็จ: Request failed \(503\)/,
		),
	).toBeTruthy();
	expect(
		screen.getByRole("button", { name: "ปิดใช้งาน Engineering Blog" }),
	).toBeTruthy();
});
