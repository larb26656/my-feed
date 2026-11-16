// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { FeedItem } from "./FeedItem";
import type { Entry } from "./types";

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
