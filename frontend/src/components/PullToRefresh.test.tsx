// @vitest-environment jsdom
import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { PullToRefresh } from "./PullToRefresh";

afterEach(cleanup);

test("refreshes only after pulling past the threshold at the top", async () => {
	const refresh = vi.fn().mockResolvedValue(undefined);
	const { container } = render(
		<PullToRefresh onRefresh={refresh}>
			<p>Feed</p>
		</PullToRefresh>,
	);
	const target = container.firstElementChild as HTMLElement;
	const touch = (y: number) => ({ touches: [{ clientX: 0, clientY: y }] });
	fireEvent.touchStart(target, touch(0));
	fireEvent.touchMove(target, touch(100));
	fireEvent.touchEnd(target);
	expect(refresh).not.toHaveBeenCalled();
	fireEvent.touchStart(target, touch(0));
	fireEvent.touchMove(target, touch(180));
	expect(screen.getByText("Release to refresh")).toBeTruthy();
	fireEvent.touchEnd(target);
	await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
});
