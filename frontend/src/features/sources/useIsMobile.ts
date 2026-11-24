import { useEffect, useState } from "react";

/** Matches the app shell breakpoint: below `md` the sidebar is hidden. */
export const sourcesMobileQuery = "(max-width: 767px)";

function mobileMatch() {
	if (
		typeof window === "undefined" ||
		typeof window.matchMedia !== "function"
	) {
		return false;
	}
	return window.matchMedia(sourcesMobileQuery).matches;
}

/**
 * True when the Sources page should mount the card list.
 * Missing `matchMedia` (unit tests, older jsdom) stays on the desktop table.
 */
export function useIsMobile() {
	const [isMobile, setIsMobile] = useState(mobileMatch);

	useEffect(() => {
		if (typeof window.matchMedia !== "function") return;
		const media = window.matchMedia(sourcesMobileQuery);
		const update = () => setIsMobile(media.matches);
		update();
		media.addEventListener("change", update);
		return () => media.removeEventListener("change", update);
	}, []);

	return isMobile;
}
