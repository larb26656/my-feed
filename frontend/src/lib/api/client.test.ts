import { afterEach, expect, test, vi } from "vitest";
import { ApiError, api } from "./client";

afterEach(() => vi.unstubAllGlobals());

test("unwraps success data and meta from the envelope", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({
				success: true,
				data: [{ id: "1" }],
				meta: { pageSize: 10, nextCursor: "abc" },
			}),
		})),
	);

	await expect(api("/entries/")).resolves.toEqual({
		data: [{ id: "1" }],
		meta: { pageSize: 10, nextCursor: "abc" },
	});
});

test("throws ApiError from a fail envelope", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async () => ({
			ok: false,
			status: 400,
			json: async () => ({
				success: false,
				error: {
					code: "VALIDATION",
					message: "Invalid link",
					detail: { field: "link" },
				},
			}),
		})),
	);

	const error = await api("/entries/save-link").catch((e) => e);
	expect(error).toBeInstanceOf(ApiError);
	expect(error).toMatchObject({
		code: "VALIDATION",
		message: "Invalid link",
		status: 400,
		detail: { field: "link" },
	});
});
