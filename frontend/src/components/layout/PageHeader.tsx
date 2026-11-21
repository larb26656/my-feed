import { Link } from "@tanstack/react-router";
import { Settings } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";
import { ThemeToggle } from "./ThemeToggle";

export function PageHeader() {
	return (
		<header className="flex h-16 items-center justify-between border-b bg-card px-5 md:hidden">
			<Link
				to="/"
				aria-label="My feed"
				className="flex items-center gap-2 font-bold"
			>
				<span className="grid size-8 place-items-center">
					<img
						src="/my-feed-logo.png"
						alt=""
						className="size-8 object-contain dark:hidden"
					/>
					<img
						src="/my-feed-logo-dark.png"
						alt=""
						className="hidden size-8 object-contain dark:block"
					/>
				</span>
			</Link>
			<div className="flex items-center gap-1">
				<ThemeToggle compact />
				<Link
					to="/settings"
					className={buttonVariants({ variant: "ghost", size: "icon" })}
					aria-label="Settings"
				>
					<Settings />
				</Link>
			</div>
		</header>
	);
}
