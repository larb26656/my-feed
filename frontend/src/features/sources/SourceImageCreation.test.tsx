// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { SourceDialog } from "./SourceDialog";
import { withSourceTypes } from "./sourceTypesTestFixture";

vi.mock("react-easy-crop", () => ({
	default: ({
		onCropComplete,
	}: {
		onCropComplete: (a: unknown, b: unknown) => void;
	}) => {
		useEffect(
			() => onCropComplete({}, { x: 0, y: 0, width: 10, height: 10 }),
			[onCropComplete],
		);
		return <div>Crop preview</div>;
	},
}));
afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});
test.each([
	"rss",
	"youtube",
] as const)("retries image upload for the %s Source already created", async (type) => {
	const user = userEvent.setup();
	vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:test");
	vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	class ImageStub {
		onload: (() => void) | null = null;
		set src(_: string) {
			this.onload?.();
		}
		async decode() {}
	}
	vi.stubGlobal("Image", ImageStub);
	vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
		drawImage: vi.fn(),
	} as unknown as CanvasRenderingContext2D);
	vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(
		(callback) => callback(new Blob(["jpeg"], { type: "image/jpeg" })),
	);
	let uploads = 0;
	const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
		if (init?.method === "POST")
			return okJson({
				id: "new-source",
				name: "News",
				type: "rss",
				active: true,
			});
		if (init?.method === "PUT")
			return ++uploads === 1
				? failJson(500, { code: "ERROR", message: "upload failed" })
				: okJson({});
		return okJson([]);
	});
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	const onClose = vi.fn();
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourceDialog editing={null} onClose={onClose} />
		</QueryClientProvider>,
	);
	await user.type(screen.getByLabelText("Source name"), "News");
	if (type === "youtube") {
		await user.click(screen.getByLabelText("YouTube"));
		await user.type(
			screen.getByLabelText("YouTube channel ID or URL"),
			"UC1234567890123456789012",
		);
	} else {
		await user.type(
			await screen.findByLabelText("RSS / Atom feed URL"),
			"https://example.com/feed",
		);
	}
	await user.upload(
		screen.getByLabelText("Choose image"),
		new File(["image"], "photo.png", { type: "image/png" }),
	);
	expect(await screen.findByText("Crop preview")).toBeTruthy();
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect((await screen.findByRole("alert")).textContent).toMatch(
		/source.*created.*image/i,
	);
	await user.click(screen.getByRole("button", { name: /retry/i }));
	await waitFor(() => expect(onClose).toHaveBeenCalled());
	expect(
		fetchMock.mock.calls.filter(([, init]) => init?.method === "POST"),
	).toHaveLength(1);
	expect(
		fetchMock.mock.calls.filter(
			([url, init]) =>
				url.includes("/sources/new-source/profile-image") &&
				init?.method === "PUT",
		),
	).toHaveLength(2);
});

test("updates preview presentation when the name and cropped image change", async () => {
	const user = userEvent.setup();
	vi.spyOn(URL, "createObjectURL")
		.mockReturnValueOnce("blob:original")
		.mockReturnValue("blob:cropped");
	vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	class ImageStub {
		onload: (() => void) | null = null;
		set src(_: string) {
			this.onload?.();
		}
		async decode() {}
	}
	vi.stubGlobal("Image", ImageStub);
	vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
		drawImage: vi.fn(),
	} as unknown as CanvasRenderingContext2D);
	vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(
		(callback) => callback(new Blob(["jpeg"], { type: "image/jpeg" })),
	);
	const fetchMock = vi.fn(async () =>
		okJson({
			entry: { title: "Preview entry", link: "https://example.test/entry" },
		}),
	);
	vi.stubGlobal("fetch", withSourceTypes(fetchMock));
	render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourceDialog editing={null} onClose={vi.fn()} />
		</QueryClientProvider>,
	);
	await user.type(
		await screen.findByLabelText("RSS / Atom feed URL"),
		"https://example.test/feed",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByRole("link", { name: "Preview entry" });
	await user.type(screen.getByLabelText("Source name"), "News");
	await user.upload(
		screen.getByLabelText("Choose image"),
		new File(["image"], "photo.png", { type: "image/png" }),
	);
	await waitFor(() =>
		expect(
			screen
				.getAllByRole("img", { name: "News avatar" })
				.some((avatar) => avatar.getAttribute("src") === "blob:cropped"),
		).toBe(true),
	);
	expect(screen.getByRole("link", { name: "Preview entry" })).toBeTruthy();
	expect(fetchMock).toHaveBeenCalledTimes(1);
});
