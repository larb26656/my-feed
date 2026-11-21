import { clearTokens, getRefreshToken, setTokens } from "./tokens";

type TokenPayload = {
	access_token: string;
	refresh_token: string;
	expires_in: number;
	token_type: string;
};

type SuccessEnvelope<T> = {
	success: true;
	data: T;
};

function authBaseURL(): string {
	return (import.meta.env.VITE_API_BASE_URL ?? "/api/v1").replace(/\/$/, "");
}

async function postAuth<T>(
	path: string,
	body: Record<string, string>,
): Promise<T> {
	const response = await fetch(`${authBaseURL()}${path}`, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(body),
	});
	const json = (await response.json()) as
		| SuccessEnvelope<T>
		| { success?: false; error?: { message?: string } };
	if (!response.ok || !json || !("success" in json) || json.success !== true) {
		const message =
			json && "error" in json && json.error?.message
				? json.error.message
				: `Request failed (${response.status})`;
		throw new Error(message);
	}
	return json.data;
}

export async function login(username: string, password: string): Promise<void> {
	const data = await postAuth<TokenPayload>("/auth/login", {
		username,
		password,
	});
	setTokens(data.access_token, data.refresh_token);
}

let refreshInFlight: Promise<boolean> | null = null;

export function refreshSession(): Promise<boolean> {
	if (!refreshInFlight) {
		refreshInFlight = performRefresh().finally(() => {
			refreshInFlight = null;
		});
	}
	return refreshInFlight;
}

async function performRefresh(): Promise<boolean> {
	const refreshToken = getRefreshToken();
	if (!refreshToken) {
		return false;
	}
	try {
		const data = await postAuth<TokenPayload>("/auth/refresh", {
			refresh_token: refreshToken,
		});
		setTokens(data.access_token, data.refresh_token);
		return true;
	} catch {
		clearTokens();
		return false;
	}
}

export async function logout(): Promise<void> {
	const refreshToken = getRefreshToken();
	if (refreshToken) {
		try {
			await postAuth<{ ok: boolean }>("/auth/logout", {
				refresh_token: refreshToken,
			});
		} catch {
			// Clear local tokens even if the server rejects a stale refresh.
		}
	}
	clearTokens();
}
