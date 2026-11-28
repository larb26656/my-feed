import { refreshSession } from "#/lib/auth/session";
import { clearTokens, getAccessToken } from "#/lib/auth/tokens";

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
	status: number;
	data: TData;
	meta?: TMeta;
};

function baseURL(): string {
	return (import.meta.env.VITE_API_BASE_URL ?? "/api/v1").replace(/\/$/, "");
}

function copyHeaders(input?: HeadersInit): Record<string, string> {
	const headers: Record<string, string> = {};
	if (!input) {
		return headers;
	}
	if (input instanceof Headers) {
		input.forEach((value, key) => {
			headers[key] = value;
		});
		return headers;
	}
	if (Array.isArray(input)) {
		for (const [key, value] of input) {
			headers[key] = value;
		}
		return headers;
	}
	return { ...input };
}

function withAuthHeaders(init?: RequestInit): RequestInit {
	const headers = copyHeaders(init?.headers);
	const hasAuthorization = Object.keys(headers).some(
		(key) => key.toLowerCase() === "authorization",
	);
	if (!hasAuthorization) {
		const accessToken = getAccessToken();
		const apiKey = import.meta.env.VITE_MY_FEED_API_KEY;
		if (accessToken) {
			headers.Authorization = `Bearer ${accessToken}`;
		} else if (apiKey) {
			headers.Authorization = `Bearer ${apiKey}`;
		}
	}
	return { ...init, headers };
}

function isAuthPath(path: string): boolean {
	return (
		path.startsWith("/auth/login") ||
		path.startsWith("/auth/refresh") ||
		path.startsWith("/auth/logout")
	);
}

export async function api<TData, TMeta = undefined>(
	path: string,
	init?: RequestInit,
	retried = false,
): Promise<ApiResult<TData, TMeta>> {
	const requestInit = withAuthHeaders(init);
	const requestToken = new Headers(requestInit.headers).get("Authorization");
	const response = await fetch(`${baseURL()}${path}`, requestInit);
	if (
		response.status === 401 &&
		!retried &&
		!isAuthPath(path) &&
		((getAccessToken() &&
			requestToken !==
				new Headers(withAuthHeaders(init).headers).get("Authorization")) ||
			(await refreshSession()))
	) {
		return api(path, init, true);
	}

	if (response.status === 401 && !isAuthPath(path)) {
		clearTokens();
		if (
			typeof window !== "undefined" &&
			window.location.pathname !== "/login"
		) {
			window.location.assign("/login");
		}
	}

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
	return { data: success.data, meta: success.meta, status: response.status };
}

/** Fetch an authenticated image as a Blob for use in an avatar. */
export async function fetchImage(path: string, retried = false): Promise<Blob> {
	const response = await fetch(`${baseURL()}${path}`, withAuthHeaders());
	if (response.status === 401 && !retried && (await refreshSession()))
		return fetchImage(path, true);
	if (!response.ok)
		throw new Error(`Could not load image (${response.status})`);
	return response.blob();
}
