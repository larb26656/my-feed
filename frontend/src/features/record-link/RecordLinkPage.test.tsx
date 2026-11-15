// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { Toaster } from "#/components/ui/toaster";
import { RecordLinkPage } from "./RecordLinkPage";

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

function renderPage(
	client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	}),
) {
	render(
		<QueryClientProvider client={client}>
			<RecordLinkPage />
			<Toaster />
		</QueryClientProvider>,
	);
	return client;
}

test("records a link, refreshes Entries, and announces success", async () => {
	const fetchMock = vi.fn(async () => ({
		ok: true,
		status: 201,
		json: async () => ({ id: "entry-1", title: "A useful article" }),
	}));
	vi.stubGlobal("fetch", fetchMock);
	const client = renderPage();
	client.setQueryData(
		["entries", { sourceId: undefined, savedOnly: false, unread: false }],
		{},
	);

	const user = userEvent.setup();
	await user.type(screen.getByLabelText("ชื่อ Entry"), "A useful article");
	await user.type(screen.getByLabelText("ลิงก์"), "https://example.com/article");
	await user.type(screen.getByLabelText("คำอธิบาย"), "A short summary");
	await user.click(screen.getByRole("button", { name: "บันทึกลิงก์" }));

	expect(await screen.findByText("บันทึกลิงก์แล้ว")).toBeTruthy();
	expect(fetchMock).toHaveBeenCalledWith(
		"/api/v1/entries/save-link",
		expect.objectContaining({
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				title: "A useful article",
				link: "https://example.com/article",
				description: "A short summary",
			}),
		}),
	);
	expect(
		client.getQueryState([
			"entries",
			{ sourceId: undefined, savedOnly: false, unread: false },
		])?.isInvalidated,
	).toBe(true);
});

test("shows field validation and submission failures", async () => {
	const fetchMock = vi.fn(async () => ({
		ok: false,
		status: 422,
		json: async () => ({}),
	}));
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(screen.getByRole("button", { name: "บันทึกลิงก์" }));
	expect(await screen.findByText("กรุณาระบุชื่อ Entry")).toBeTruthy();
	expect(await screen.findByText("กรุณาระบุลิงก์ที่ถูกต้อง")).toBeTruthy();
	expect(fetchMock).not.toHaveBeenCalled();
	await user.type(screen.getByLabelText("ชื่อ Entry"), "A useful article");
	await user.type(screen.getByLabelText("ลิงก์"), "https://example.com/article");
	await user.click(screen.getByRole("button", { name: "บันทึกลิงก์" }));
	expect((await screen.findByRole("alert")).textContent).toContain(
		"บันทึกลิงก์ไม่สำเร็จ: Request failed (422)",
	);
	expect(
		await screen.findByRole("heading", { name: "บันทึกลิงก์ไม่สำเร็จ" }),
	).toBeTruthy();
});

test("shows a field error for an invalid publication date without submitting", async () => {
	const fetchMock = vi.fn();
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.type(screen.getByLabelText("ชื่อ Entry"), "A useful article");
	await user.type(screen.getByLabelText("ลิงก์"), "https://example.com/article");
	const publishedAt = screen.getByLabelText("วันที่เผยแพร่ (ไม่บังคับ)");
	publishedAt.setAttribute("type", "text");
	fireEvent.change(publishedAt, { target: { value: "not-a-date" } });
	await user.click(screen.getByRole("button", { name: "บันทึกลิงก์" }));

	expect(await screen.findByText("กรุณาระบุวันที่เผยแพร่ที่ถูกต้อง")).toBeTruthy();
	expect(fetchMock).not.toHaveBeenCalled();
});
