// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { useSources } from "./useSources";

afterEach(() => vi.unstubAllGlobals());

test("loads sources beyond the first page", async () => {
	const fetch = vi.fn().mockImplementation(async (url: string) => ({
		ok: true,
		json: async () => ({
			success: true,
			data: [{ id: url.includes("page=2") ? "later" : "first" }],
			meta: { totalPages: 2 },
		}),
	}));
	vi.stubGlobal("fetch", fetch);
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	const wrapper = ({ children }: { children: ReactNode }) => (
		<QueryClientProvider client={client}>{children}</QueryClientProvider>
	);
	const { result } = renderHook(() => useSources(), { wrapper });
	await waitFor(() => expect(result.current.isSuccess).toBe(true));
	expect(result.current.data?.map((source) => source.id)).toEqual([
		"first",
		"later",
	]);
	expect(fetch).toHaveBeenCalledTimes(2);
});
