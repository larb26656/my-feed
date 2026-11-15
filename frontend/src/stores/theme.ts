import { create } from "zustand";
import { persist } from "zustand/middleware";

type Theme = "light" | "dark";

type ThemeState = {
	theme: Theme;
	toggleTheme: () => void;
};

export const useThemeStore = create<ThemeState>()(
	persist(
		(set) => ({
			theme:
				typeof window !== "undefined" &&
				window.matchMedia?.("(prefers-color-scheme: dark)").matches
					? "dark"
					: "light",
			toggleTheme: () =>
				set((state) => ({
					theme: state.theme === "dark" ? "light" : "dark",
				})),
		}),
		{
			name: "my-feed-theme",
			partialize: (state) => ({ theme: state.theme }),
		},
	),
);

// Apply the persisted theme before React renders, then keep the document in sync.
export function initializeTheme() {
	const applyTheme = ({ theme }: ThemeState) => {
		document.documentElement.classList.toggle("dark", theme === "dark");
	};
	applyTheme(useThemeStore.getState());
	return useThemeStore.subscribe(applyTheme);
}
