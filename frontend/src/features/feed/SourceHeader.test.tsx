// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
				actions={{
					activationPending: false,
					onEdit: vi.fn(),
					onImage: vi.fn(),
					onToggleActive: vi.fn(),
				}}
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
		screen.getByRole("button", { name: "Actions for Fireship" }),
	).toBeTruthy();
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
	expect(screen.queryByRole("button", { name: /Actions for/ })).toBeNull();
});

test("opens the shared source actions and invokes edit", async () => {
	const onEdit = vi.fn();
	const client = new QueryClient();
	render(
		<QueryClientProvider client={client}>
			<SourceHeader
				source={fireship}
				loading={false}
				error={false}
				actions={{
					activationPending: false,
					onEdit,
					onImage: vi.fn(),
					onToggleActive: vi.fn(),
				}}
			/>
		</QueryClientProvider>,
	);
	await userEvent.click(
		screen.getByRole("button", { name: "Actions for Fireship" }),
	);
	await userEvent.click(
		await screen.findByRole("menuitem", { name: "Edit source" }),
	);
	expect(onEdit).toHaveBeenCalledWith(fireship);
});
