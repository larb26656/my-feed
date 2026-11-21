// @vitest-environment jsdom
import { afterEach, expect, test, vi } from "vitest";
import { clearTokens, getAccessToken, setTokens } from "#/lib/auth/tokens";
import { ApiError, api } from "./client";

afterEach(() => {
	vi.unstubAllGlobals();
	vi.unstubAllEnvs();
	clearTokens();
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

test("attaches access token from localStorage before API key", async () => {
	setTokens("access-from-login", "refresh");
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
		"Bearer access-from-login",
	);
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

test("omits Authorization when no access token or API key", async () => {
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

test("refreshes once on 401 then retries", async () => {
	setTokens("stale-access", "refresh-token");
	let entriesCalls = 0;
	const fetchMock = vi.fn(async (url: string) => {
		if (String(url).includes("/auth/refresh")) {
			return {
				ok: true,
				status: 200,
				json: async () => ({
					success: true,
					data: {
						access_token: "new-access",
						refresh_token: "new-refresh",
						expires_in: 900,
						token_type: "Bearer",
					},
				}),
			};
		}
		entriesCalls += 1;
		if (entriesCalls === 1) {
			return {
				ok: false,
				status: 401,
				json: async () => ({
					success: false,
					error: { code: "UNAUTHORIZED", message: "expired" },
				}),
			};
		}
		return {
			ok: true,
			status: 200,
			json: async () => ({ success: true, data: [{ id: "1" }] }),
		};
	});
	vi.stubGlobal("fetch", fetchMock);

	await expect(api("/entries/")).resolves.toEqual({
		data: [{ id: "1" }],
		meta: undefined,
	});
	expect(getAccessToken()).toBe("new-access");
	expect(fetchMock).toHaveBeenCalledTimes(3);
	const retryInit = fetchMock.mock.calls[2]?.[1] as RequestInit | undefined;
	expect(new Headers(retryInit?.headers).get("Authorization")).toBe(
		"Bearer new-access",
	);
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

test("concurrent expired requests share one refresh and both retry", async () => {
	setTokens("stale-access", "refresh-token");
	let refreshCalls = 0;
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (String(url).includes("/auth/refresh")) {
			refreshCalls += 1;
			await new Promise((resolve) => setTimeout(resolve, 10));
			return {
				ok: true,
				status: 200,
				json: async () => ({
					success: true,
					data: {
						access_token: "new-access",
						refresh_token: "new-refresh",
						expires_in: 900,
						token_type: "Bearer",
					},
				}),
			};
		}
		if (
			new Headers(init?.headers).get("Authorization") === "Bearer stale-access"
		) {
			return { ok: false, status: 401, json: async () => ({ success: false }) };
		}
		return {
			ok: true,
			status: 200,
			json: async () => ({ success: true, data: [] }),
		};
	});
	vi.stubGlobal("fetch", fetchMock);
	await Promise.all([api("/entries/"), api("/sources/")]);
	expect(refreshCalls).toBe(1);
});
