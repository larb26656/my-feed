// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, test } from "vitest";
import { initializeTheme, useThemeStore } from "#/stores/theme";
import { ThemeToggle } from "./ThemeToggle";

let unsubscribe: () => void;

beforeEach(() => {
	useThemeStore.setState({ theme: "light" });
	localStorage.clear();
	unsubscribe = initializeTheme();
});

afterEach(() => {
	cleanup();
	unsubscribe();
	localStorage.clear();
	document.documentElement.classList.remove("dark");
});

test("desktop and mobile toggles share the theme and update the whole document", async () => {
	render(
		<>
			<ThemeToggle />
			<ThemeToggle compact />
		</>,
	);
	const user = userEvent.setup();
	await user.click(
		screen.getAllByRole("button", { name: "Switch to dark mode" })[0],
	);
	expect(document.documentElement.classList.contains("dark")).toBe(true);
	expect(
		screen.getAllByRole("button", { name: "Switch to light mode" }),
	).toHaveLength(2);
	expect(
		JSON.parse(localStorage.getItem("my-feed-theme") ?? "{}").state.theme,
	).toBe("dark");
	await user.click(
		screen.getAllByRole("button", { name: "Switch to light mode" })[1],
	);
	expect(document.documentElement.classList.contains("dark")).toBe(false);
	expect(
		screen.getAllByRole("button", { name: "Switch to dark mode" }),
	).toHaveLength(2);
});

test("restores a saved preference before the theme controls mount", async () => {
	localStorage.setItem(
		"my-feed-theme",
		JSON.stringify({ state: { theme: "dark" }, version: 0 }),
	);
	await useThemeStore.persist.rehydrate();
	unsubscribe();
	document.documentElement.classList.remove("dark");
	unsubscribe = initializeTheme();
	expect(document.documentElement.classList.contains("dark")).toBe(true);
	render(<ThemeToggle />);
	expect(
		screen.getByRole("button", { name: "Switch to light mode" }),
	).toBeTruthy();
});
