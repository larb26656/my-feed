// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { ProfileImageDialog } from "./ProfileImageDialog";

vi.mock("react-easy-crop", () => ({
	default: ({
		onCropComplete,
	}: {
		onCropComplete: (area: unknown, pixels: unknown) => void;
	}) => {
		useEffect(() => {
			onCropComplete({}, { x: 0, y: 0, width: 10, height: 10 });
		}, [onCropComplete]);
		return <div>Crop preview</div>;
	},
}));
afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});
test("selects and crops a profile image with a labelled zoom control", async () => {
	const user = userEvent.setup();
	vi.stubGlobal("URL", {
		createObjectURL: () => "blob:test",
		revokeObjectURL: vi.fn(),
	});
	class ImageStub {
		onload: (() => void) | null = null;
		onerror: (() => void) | null = null;
		set src(_: string) {
			this.onload?.();
		}
	}
	vi.stubGlobal("Image", ImageStub);
	const client = new QueryClient();
	render(
		<QueryClientProvider client={client}>
			<ProfileImageDialog
				source={{ id: "one", name: "News", type: "rss", active: true }}
				onClose={() => {}}
			/>
		</QueryClientProvider>,
	);
	await user.upload(
		screen.getByLabelText("Choose image"),
		new File(["image"], "photo.png", { type: "image/png" }),
	);
	expect(await screen.findByText("Crop preview")).toBeTruthy();
	const zoom = screen.getByLabelText("Zoom") as HTMLInputElement;
	fireEvent.change(zoom, { target: { value: "2" } });
	expect(Number(zoom.value)).toBe(2);
	expect(screen.getByRole("button", { name: "Save image" })).not.toHaveProperty(
		"disabled",
		true,
	);
});

test("removes a profile image independently of source details", async () => {
	const user = userEvent.setup();
	const fetchMock = vi.fn().mockImplementation(() =>
		Promise.resolve(
			new Response(JSON.stringify({ success: true, data: {} }), {
				status: 200,
			}),
		),
	);
	vi.stubGlobal("fetch", fetchMock);
	const onClose = vi.fn();
	render(
		<QueryClientProvider client={new QueryClient()}>
			<ProfileImageDialog
				source={{
					id: "one",
					name: "News",
					type: "rss",
					active: false,
					profileImageId: "image",
				}}
				onClose={onClose}
			/>
		</QueryClientProvider>,
	);
	await user.click(screen.getByRole("button", { name: "Remove image" }));
	expect(fetchMock).toHaveBeenCalledWith(
		"/api/v1/sources/one/profile-image",
		expect.objectContaining({ method: "DELETE" }),
	);
	await waitFor(() => expect(onClose).toHaveBeenCalled());
});
