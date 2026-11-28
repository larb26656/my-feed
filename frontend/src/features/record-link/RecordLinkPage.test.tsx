// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { Toaster } from "#/components/ui/toaster";
import { failJson, okJson } from "#/lib/api/test-helpers";
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
	const fetchMock = vi.fn(async () =>
		okJson({ id: "entry-1", title: "A useful article" }, undefined, 201),
	);
	vi.stubGlobal("fetch", fetchMock);
	const client = renderPage();
	client.setQueryData(
		["entries", { sourceId: undefined, savedOnly: false, unread: false }],
		{},
	);

	const user = userEvent.setup();
	await user.type(screen.getByLabelText("Entry title"), "A useful article");
	await user.type(
		screen.getByLabelText("Link (optional)"),
		"https://example.com/article",
	);
	await user.type(screen.getByLabelText("Description"), "A short summary");
	await user.click(screen.getByRole("button", { name: "Save Link" }));

	expect(await screen.findByText("Link saved")).toBeTruthy();
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
	const fetchMock = vi.fn(async () => failJson(422));
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(screen.getByRole("button", { name: "Save Link" }));
	expect(await screen.findByText("Please enter an entry title")).toBeTruthy();
	expect(fetchMock).not.toHaveBeenCalled();
	await user.type(screen.getByLabelText("Entry title"), "A useful article");
	await user.type(
		screen.getByLabelText("Link (optional)"),
		"https://example.com/article",
	);
	await user.click(screen.getByRole("button", { name: "Save Link" }));
	expect((await screen.findByRole("alert")).textContent).toContain(
		"Could not save link: Request failed (422)",
	);
	expect(
		await screen.findByRole("heading", { name: "Could not save link" }),
	).toBeTruthy();
});

test("shows a field error for an invalid publication date without submitting", async () => {
	const fetchMock = vi.fn();
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.type(screen.getByLabelText("Entry title"), "A useful article");
	await user.type(
		screen.getByLabelText("Link (optional)"),
		"https://example.com/article",
	);
	const publishedAt = screen.getByLabelText("Publication date (optional)");
	publishedAt.setAttribute("type", "text");
	fireEvent.change(publishedAt, { target: { value: "not-a-date" } });
	await user.click(screen.getByRole("button", { name: "Save Link" }));

	expect(
		await screen.findByText("Please enter a valid publication date"),
	).toBeTruthy();
	expect(fetchMock).not.toHaveBeenCalled();
});

test("records an Entry without a link", async () => {
	const fetchMock = vi.fn(async () =>
		okJson({ id: "entry-1", title: "Standalone", link: null }, undefined, 201),
	);
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.type(screen.getByLabelText("Entry title"), "Standalone");
	await user.click(screen.getByRole("button", { name: "Save Link" }));
	expect(await screen.findByText("Link saved")).toBeTruthy();
	expect(fetchMock).toHaveBeenCalledWith(
		"/api/v1/entries/save-link",
		expect.objectContaining({
			body: JSON.stringify({ title: "Standalone", link: "", description: "" }),
		}),
	);
});
