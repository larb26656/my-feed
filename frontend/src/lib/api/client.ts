export class ApiError extends Error {
	readonly code: string;
	readonly status: number;
	readonly detail?: unknown;

	constructor(code: string, message: string, status: number, detail?: unknown) {
		super(message);
		this.name = "ApiError";
		this.code = code;
		this.status = status;
		this.detail = detail;
	}
}

export function isApiError(error: unknown): error is ApiError {
	return error instanceof ApiError;
}

type SuccessEnvelope<TData, TMeta = unknown> = {
	success: true;
	data: TData;
	meta?: TMeta;
};

type FailEnvelope = {
	success?: false;
	error?: {
		code?: string;
		message?: string;
		detail?: unknown;
	};
};

export type ApiResult<TData, TMeta = undefined> = {
	data: TData;
	meta?: TMeta;
};

function baseURL(): string {
	return (import.meta.env.VITE_API_BASE_URL ?? "/api/v1").replace(/\/$/, "");
}

export async function api<TData, TMeta = undefined>(
	path: string,
	init?: RequestInit,
): Promise<ApiResult<TData, TMeta>> {
	const response = await fetch(`${baseURL()}${path}`, init);
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		throw new ApiError(
			"UNKNOWN",
			`Request failed (${response.status})`,
			response.status,
		);
	}

	if (
		typeof body === "object" &&
		body !== null &&
		"success" in body &&
		(body as { success: unknown }).success === false
	) {
		const fail = body as FailEnvelope;
		throw new ApiError(
			fail.error?.code ?? "UNKNOWN",
			fail.error?.message ?? `Request failed (${response.status})`,
			response.status,
			fail.error?.detail,
		);
	}

	if (!response.ok) {
		throw new ApiError(
			"UNKNOWN",
			`Request failed (${response.status})`,
			response.status,
		);
	}

	if (
		typeof body !== "object" ||
		body === null ||
		!("success" in body) ||
		(body as { success: unknown }).success !== true ||
		!("data" in body)
	) {
		throw new ApiError(
			"UNKNOWN",
			`Unexpected success response (${response.status})`,
			response.status,
		);
	}

	const success = body as SuccessEnvelope<TData, TMeta>;
	return { data: success.data, meta: success.meta };
}
