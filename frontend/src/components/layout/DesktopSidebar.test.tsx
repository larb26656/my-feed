// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	cleanup,
	screen,
	render as testingRender,
	within,
} from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { DesktopSidebar } from "./DesktopSidebar";

const feedState = vi.hoisted(() => ({
	sources: [] as Array<{
		id: string;
		name: string;
		image?: string;
		active: boolean;
	}>,
	savedCount: 0 as number | undefined,
	savedCountError: false,
}));

vi.mock("@tanstack/react-router", () => ({
	useRouterState: ({
		select,
	}: {
		select: (state: { location: { pathname: string } }) => string;
	}) => select({ location: { pathname: "/" } }),
	Link: ({
		to,
		params,
		activeOptions: _activeOptions,
		children,
		...props
	}: ComponentProps<"a"> & {
		to: string;
		params?: { sourceId?: string };
		activeOptions?: unknown;
	}) => {
		const href =
			to === "/sources/$sourceId" && params?.sourceId
				? `/sources/${params.sourceId}`
				: to;
		return (
			<a href={href} {...props}>
				{children}
			</a>
		);
	},
}));

vi.mock("#/features/feed/FeedState", () => ({
	useFeedState: () => feedState,
}));

vi.mock("#/hooks/queries", () => ({
	useSidebarSources: () => ({ data: feedState.sources }),
}));

vi.mock("#/stores/theme", () => ({
	useThemeStore: (
		select: (state: { theme: string; toggleTheme: () => void }) => unknown,
	) => select({ theme: "light", toggleTheme: () => undefined }),
}));

function render(ui: React.ReactElement) {
	const client = new QueryClient();
	return testingRender(
		<QueryClientProvider client={client}>{ui}</QueryClientProvider>,
	);
}

afterEach(() => {
	cleanup();
	feedState.sources = [];
	feedState.savedCount = 0;
	feedState.savedCountError = false;
});

test("puts Manage Sources in the main menu and lists sources returned by the sidebar query", () => {
	feedState.sources = [
		...Array.from({ length: 10 }, (_, index) => ({
			id: `source-${index + 1}`,
			name: `Source ${index + 1}`,
			active: true,
		})),
	];

	render(<DesktopSidebar />);

	const mainNav = screen.getByRole("navigation", { name: "Main navigation" });
	expect(
		within(mainNav)
			.getByRole("link", { name: /Manage Sources/ })
			.getAttribute("href"),
	).toBe("/sources");

	const sourceHrefs = screen
		.getAllByRole("link")
		.map((link) => link.getAttribute("href"))
		.filter((href) => href?.startsWith("/sources/source-"));
	expect(sourceHrefs).toEqual(
		Array.from({ length: 10 }, (_, index) => `/sources/source-${index + 1}`),
	);
});
