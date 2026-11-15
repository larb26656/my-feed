import { afterEach, expect, test, vi } from "vitest";
import { api } from "#/lib/api";

afterEach(() => vi.unstubAllGlobals());

test("surfaces the backend error message for a failed API response", async () => {
	vi.stubGlobal(
		"fetch",
		vi.fn(async () => ({
			ok: false,
			status: 422,
			json: async () => ({ error: "ลิงก์ไม่ถูกต้อง", code: "INVALID_LINK" }),
		})),
	);

	await expect(api("/entries/save-link")).rejects.toThrow("ลิงก์ไม่ถูกต้อง");
});
