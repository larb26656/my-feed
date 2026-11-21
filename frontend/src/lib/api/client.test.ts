import { afterEach, expect, test, vi } from "vitest";
import { ApiError, api } from "./client";

afterEach(() => {
	vi.unstubAllGlobals();
	vi.unstubAllEnvs();
});

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

test("attaches Bearer token when VITE_MY_FEED_API_KEY is set", async () => {
	vi.stubEnv("VITE_MY_FEED_API_KEY", "dev-secret");
	const fetchMock = vi.fn(async () => ({
		ok: true,
		status: 200,
		json: async () => ({ success: true, data: [] }),
	}));
	vi.stubGlobal("fetch", fetchMock);

	await api("/entries/");

	const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
	expect(new Headers(init?.headers).get("Authorization")).toBe(
		"Bearer dev-secret",
	);
});

test("omits Authorization when VITE_MY_FEED_API_KEY is unset", async () => {
	vi.stubEnv("VITE_MY_FEED_API_KEY", "");
	const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
		expect(new Headers(init?.headers).has("Authorization")).toBe(false);
		return {
			ok: true,
			status: 200,
			json: async () => ({ success: true, data: [] }),
		};
	});
	vi.stubGlobal("fetch", fetchMock);

	await api("/entries/");
	expect(fetchMock).toHaveBeenCalled();
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
