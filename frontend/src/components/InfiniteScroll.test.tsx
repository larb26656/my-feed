// @vitest-environment jsdom
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { InfiniteScroll } from "./InfiniteScroll";

let callback: IntersectionObserverCallback;
const disconnect = vi.fn();
class Observer {
	constructor(cb: IntersectionObserverCallback) {
		callback = cb;
	}
	observe = vi.fn();
	disconnect = disconnect;
}
const intersect = () =>
	callback(
		[{ isIntersecting: true } as IntersectionObserverEntry],
		{} as IntersectionObserver,
	);
afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
	vi.clearAllMocks();
});

test("loads once for repeated intersections while pending and disconnects", async () => {
	vi.stubGlobal("IntersectionObserver", Observer);
	let finish = () => {};
	const load = vi.fn(
		() =>
			new Promise<void>((resolve) => {
				finish = resolve;
			}),
	);
	const view = render(
		<InfiniteScroll
			next={{ hasPage: true, isLoading: false, isError: false, load }}
		>
			<p>Items</p>
		</InfiniteScroll>,
	);
	act(() => {
		intersect();
		intersect();
	});
	expect(load).toHaveBeenCalledTimes(1);
	expect(screen.getByText("Items")).toBeTruthy();
	await act(async () => finish());
	view.unmount();
	expect(disconnect).toHaveBeenCalled();
});

test("stops observing on error and lets the user retry", async () => {
	vi.stubGlobal("IntersectionObserver", Observer);
	const load = vi.fn(async () => {});
	render(
		<InfiniteScroll
			next={{ hasPage: true, isLoading: false, isError: true, load }}
		>
			Items
		</InfiniteScroll>,
	);
	expect(screen.getByRole("alert")).toBeTruthy();
	expect(load).not.toHaveBeenCalled();
	await userEvent.click(screen.getByRole("button", { name: "Try again" }));
	expect(load).toHaveBeenCalledTimes(1);
});

test("offers manual loading when IntersectionObserver is unavailable", async () => {
	vi.stubGlobal("IntersectionObserver", undefined);
	const load = vi.fn(async () => {});
	const { rerender } = render(
		<InfiniteScroll
			next={{ hasPage: true, isLoading: false, isError: false, load }}
		>
			Items
		</InfiniteScroll>,
	);
	await userEvent.click(screen.getByRole("button", { name: "Load more" }));
	expect(load).toHaveBeenCalledTimes(1);
	rerender(
		<InfiniteScroll
			next={{ hasPage: false, isLoading: false, isError: false, load }}
		>
			Items
		</InfiniteScroll>,
	);
	expect(screen.queryByRole("button", { name: "Load more" })).toBeNull();
});
