// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { SourceHeader } from "./SourceHeader";
import type { Source } from "./types";

vi.mock("@tanstack/react-router", () => ({
	Link: ({
		children,
		to,
		search,
	}: {
		children: React.ReactNode;
		to?: string;
		search?: { sourceId?: string };
	}) => (
		<a href={search?.sourceId ? `${to}?sourceId=${search.sourceId}` : to}>
			{children}
		</a>
	),
}));

afterEach(() => {
	cleanup();
});

function renderHeader(
	source: Source | undefined,
	options: { loading?: boolean; error?: boolean } = {},
) {
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	return render(
		<QueryClientProvider client={client}>
			<SourceHeader
				source={source}
				loading={options.loading ?? false}
				error={options.error ?? false}
			/>
		</QueryClientProvider>,
	);
}

const fireship: Source = {
	id: "fireship",
	name: "Fireship",
	type: "youtube",
	active: true,
	description: "Videos about web, AI, and modern dev.",
};

test("centers the source name, type, and description", () => {
	renderHeader(fireship);
	expect(screen.getByRole("heading", { name: "Fireship" })).toBeTruthy();
	expect(screen.getByText("YouTube")).toBeTruthy();
	expect(
		screen.getByText("Videos about web, AI, and modern dev."),
	).toBeTruthy();
	expect(
		screen.getByRole("link", { name: "Manage sources" }).getAttribute("href"),
	).toBe("/sources?sourceId=fireship");
});

test("hides an empty description and labels other source types", () => {
	const { rerender } = renderHeader({
		id: "blog",
		name: "Engineering Blog",
		type: "rss",
		active: false,
		description: "   ",
	});
	expect(screen.getByText("RSS")).toBeTruthy();
	expect(screen.getByText("Inactive")).toBeTruthy();
	expect(
		screen.queryByText("Videos about web, AI, and modern dev."),
	).toBeNull();

	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	rerender(
		<QueryClientProvider client={client}>
			<SourceHeader
				source={{
					id: "saved",
					name: "Saved links",
					type: "manual",
					active: true,
				}}
				loading={false}
				error={false}
			/>
		</QueryClientProvider>,
	);
	expect(screen.getByRole("heading", { name: "Saved links" })).toBeTruthy();
	expect(screen.getByText("Manual")).toBeTruthy();
});

test("shows a loading state and an unavailable source message", () => {
	const view = renderHeader(undefined, { loading: true });
	expect(screen.getByText("Loading source")).toBeTruthy();
	expect(screen.queryByRole("link", { name: "Manage sources" })).toBeNull();
	view.rerender(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourceHeader loading={false} error />
		</QueryClientProvider>,
	);
	expect(screen.getByText("Could not load source details.")).toBeTruthy();
	expect(
		screen.getByRole("link", { name: "Manage sources" }).getAttribute("href"),
	).toBe("/sources");
});
