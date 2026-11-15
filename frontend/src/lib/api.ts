export async function api<T>(path: string, init?: RequestInit): Promise<T> {
	const baseURL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";
	const response = await fetch(`${baseURL.replace(/\/$/, "")}${path}`, init);
	if (!response.ok) {
		let message = `Request failed (${response.status})`;
		try {
			const body: unknown = await response.json();
			if (
				typeof body === "object" &&
				body !== null &&
				"error" in body &&
				typeof body.error === "string" &&
				body.error.trim()
			) {
				message = body.error;
			}
		} catch {
			// Keep the status fallback when the response has no readable JSON body.
		}
		throw new Error(message);
	}
	return response.json() as Promise<T>;
}
