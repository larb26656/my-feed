const ACCESS_KEY = "my-feed.access_token";
const REFRESH_KEY = "my-feed.refresh_token";

function storage(): Storage | null {
	if (typeof localStorage === "undefined") {
		return null;
	}
	return localStorage;
}

export function getAccessToken(): string | null {
	return storage()?.getItem(ACCESS_KEY) ?? null;
}

export function getRefreshToken(): string | null {
	return storage()?.getItem(REFRESH_KEY) ?? null;
}

export function setTokens(accessToken: string, refreshToken: string): void {
	const store = storage();
	if (!store) {
		return;
	}
	store.setItem(ACCESS_KEY, accessToken);
	store.setItem(REFRESH_KEY, refreshToken);
}

export function clearTokens(): void {
	const store = storage();
	if (!store) {
		return;
	}
	store.removeItem(ACCESS_KEY);
	store.removeItem(REFRESH_KEY);
}

export function hasSessionTokens(): boolean {
	return Boolean(getAccessToken() || getRefreshToken());
}
