// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { SourceAvatar } from "./SourceAvatar";

afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});

test("shows the uploaded image and falls back when it cannot be displayed", async () => {
	const revoke = vi.fn();
	vi.stubGlobal("URL", {
		createObjectURL: () => "blob:avatar",
		revokeObjectURL: revoke,
	});
	vi.stubGlobal(
		"fetch",
		vi
			.fn()
			.mockResolvedValue({ ok: true, blob: async () => new Blob(["image"]) }),
	);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	const view = render(
		<QueryClientProvider client={client}>
			<SourceAvatar name="News" sourceId="one" imageId="image" />
		</QueryClientProvider>,
	);
	const avatar = screen.getByRole("img", { name: "News avatar" });
	await waitFor(() => expect(avatar.getAttribute("src")).toBe("blob:avatar"));
	fireEvent.error(avatar);
	expect(avatar.getAttribute("src")).toMatch(/^data:image\/svg\+xml/);
	view.unmount();
	expect(revoke).toHaveBeenCalledWith("blob:avatar");
});
