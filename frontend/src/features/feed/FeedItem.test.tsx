// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	cleanup,
	screen,
	render as testingRender,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { FeedItem } from "./FeedItem";
import type { Entry } from "./types";

vi.mock("@tanstack/react-router", () => ({
	Link: ({
		children,
		params,
		...props
	}: {
		children: React.ReactNode;
		params: { sourceId: string };
	}) => (
		<a href={`/sources/${params.sourceId}`} {...props}>
			{children}
		</a>
	),
}));

function render(ui: React.ReactElement) {
	const client = new QueryClient();
	return testingRender(
		<QueryClientProvider client={client}>{ui}</QueryClientProvider>,
	);
}

afterEach(cleanup);

const entry: Entry = {
	id: "entry",
	sourceId: "source",
	title: "Entry",
	link: "https://example.com/page",
	description: "",
	createdAt: "2025-01-01T00:00:00Z",
};

test("displays the supplied thumbnail linked to the reading page", () => {
	const { container } = render(
		<FeedItem
			entry={{ ...entry, thumbnail: "https://example.com/image.png" }}
			disabled={false}
			onToggle={vi.fn()}
		/>,
	);
	const image = container.querySelector(
		'img[src="https://example.com/image.png"]',
	);
	expect(image).not.toBeNull();
	expect(image?.closest("a")?.getAttribute("href")).toBe(entry.link);
});

test("keeps an Entry readable without a thumbnail", () => {
	const { container } = render(
		<FeedItem entry={entry} disabled={false} onToggle={vi.fn()} />,
	);
	expect(screen.getByRole("link", { name: "Entry" }).getAttribute("href")).toBe(
		entry.link,
	);
	expect(container.querySelector('img[alt=""]')).toBeNull();
});

test("links the source name and avatar to its feed", () => {
	render(
		<FeedItem
			entry={entry}
			source={{ id: "source", name: "Example", type: "rss", active: true }}
			disabled={false}
			onToggle={vi.fn()}
		/>,
	);
	expect(
		screen.getByRole("link", { name: "Example" }).getAttribute("href"),
	).toBe("/sources/source");
	expect(
		screen
			.getByRole("link", { name: "View Example feed" })
			.getAttribute("href"),
	).toBe("/sources/source");
});
