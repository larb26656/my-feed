// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { MobileNavigation } from "./MobileNavigation";

const location = vi.hoisted(() => ({ pathname: "/" }));
vi.mock("@tanstack/react-router", () => ({
	useLocation: ({ select }: { select: (value: typeof location) => string }) =>
		select(location),
	Link: ({
		to,
		activeOptions: _activeOptions,
		...props
	}: ComponentProps<"a"> & { to: string; activeOptions?: unknown }) => (
		<a href={to} {...props} />
	),
}));

beforeEach(() => {
	location.pathname = "/";
	vi.stubGlobal("scrollY", 0);
	vi.stubGlobal("innerHeight", 600);
	vi.spyOn(document.documentElement, "scrollHeight", "get").mockReturnValue(
		2000,
	);
});
afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});
function scrollTo(y: number) {
	vi.stubGlobal("scrollY", y);
	fireEvent.scroll(window);
}
function hidden() {
	return screen.getByRole("navigation").classList.contains("opacity-0");
}

test("hides while reading down and reveals after small cumulative upward movement", () => {
	render(<MobileNavigation />);
	scrollTo(100);
	expect(hidden()).toBe(true);
	scrollTo(98);
	scrollTo(96);
	expect(hidden()).toBe(true);
	scrollTo(94);
	expect(hidden()).toBe(false);
	scrollTo(104);
	scrollTo(117);
	expect(hidden()).toBe(false);
	scrollTo(118);
	expect(hidden()).toBe(true);
	scrollTo(20);
	expect(hidden()).toBe(false);
});

test("ignores elastic overscroll until the reader scrolls back into the feed", () => {
	render(<MobileNavigation />);
	scrollTo(1400);
	expect(hidden()).toBe(true);
	scrollTo(1460);
	scrollTo(1400);
	expect(hidden()).toBe(true);
	scrollTo(1394);
	expect(hidden()).toBe(false);
});

test("keyboard focus reveals hidden links while retaining their tab order", () => {
	render(<MobileNavigation />);
	scrollTo(100);
	expect(hidden()).toBe(true);
	const saved = screen.getByRole("link", { name: "Saved" });
	fireEvent.focus(saved);
	expect(hidden()).toBe(false);
	expect(saved.tabIndex).toBe(0);
});

test("shows only the current label and resets visibility when changing destinations", () => {
	const { rerender } = render(<MobileNavigation />);
	expect(screen.getByText("My Feed")).toBeTruthy();
	expect(screen.queryByText("Saved")).toBeNull();
	expect(screen.getAllByRole("link")).toHaveLength(3);
	scrollTo(100);
	expect(hidden()).toBe(true);
	location.pathname = "/sources/example";
	rerender(<MobileNavigation />);
	expect(hidden()).toBe(false);
	expect(screen.getByText("Sources")).toBeTruthy();
	expect(
		screen.getByRole("link", { name: "Sources" }).getAttribute("aria-current"),
	).toBe("page");
	expect(
		screen.getByRole("link", { name: "My Feed" }).getAttribute("aria-current"),
	).toBeNull();
	expect(screen.getByRole("link", { name: "Saved" }).getAttribute("href")).toBe(
		"/saved",
	);
});

test("removes the scroll listener on unmount", () => {
	const remove = vi.spyOn(window, "removeEventListener");
	const { unmount } = render(<MobileNavigation />);
	unmount();
	expect(remove).toHaveBeenCalledWith("scroll", expect.any(Function));
});
