/** Build a successful API JSON response matching the backend envelope. */
export function okJson(data: unknown, meta?: unknown, status = 200) {
	return {
		ok: true as const,
		status,
		json: async () =>
			meta === undefined
				? { success: true, data }
				: { success: true, data, meta },
	};
}

/** Build a failed API JSON response matching the backend envelope. */
export function failJson(
	status: number,
	error: { code?: string; message?: string; detail?: unknown } = {},
) {
	return {
		ok: false as const,
		status,
		json: async () => ({
			success: false,
			error: {
				code: error.code ?? "UNKNOWN",
				message: error.message ?? `Request failed (${status})`,
				detail: error.detail,
			},
		}),
	};
}
