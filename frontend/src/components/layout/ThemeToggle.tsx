import { Moon, Sun } from "lucide-react";
import { Button } from "#/components/ui/button";
import { useThemeStore } from "#/stores/theme";

export function ThemeToggle({ compact = false }: { compact?: boolean }) {
	const theme = useThemeStore((state) => state.theme);
	const toggleTheme = useThemeStore((state) => state.toggleTheme);
	const label =
		theme === "dark" ? "Switch to light mode" : "Switch to dark mode";
	const Icon = theme === "dark" ? Sun : Moon;

	return (
		<Button
			type="button"
			variant="ghost"
			size={compact ? "icon" : "default"}
			className={compact ? undefined : "h-11 w-full justify-start"}
			onClick={toggleTheme}
			aria-label={label}
			title={label}
		>
			<Icon data-icon="inline-start" />
			{!compact && (theme === "dark" ? "Light mode" : "Dark mode")}
		</Button>
	);
}
