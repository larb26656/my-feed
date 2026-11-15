const minute = 60_000;
const hour = 60 * minute;

/** Formats a timestamp relative to the current local day and time. */
export function formatDate(value: string | Date, now = new Date()): string {
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return "Unknown date";

	const elapsed = now.getTime() - date.getTime();
	if (elapsed >= 0 && elapsed < hour) {
		const minutes = Math.floor(elapsed / minute);
		return minutes < 1 ? "Just now" : `${minutes}m ago`;
	}

	const sameDay =
		date.getFullYear() === now.getFullYear() &&
		date.getMonth() === now.getMonth() &&
		date.getDate() === now.getDate();

	if (sameDay) {
		return date.toLocaleTimeString(undefined, {
			hour: "numeric",
			minute: "2-digit",
		});
	}

	return date.toLocaleString(undefined, {
		year: "numeric",
		month: "numeric",
		day: "numeric",
		hour: "numeric",
		minute: "2-digit",
	});
}
