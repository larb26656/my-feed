import { Link } from "@tanstack/react-router";
import { Settings } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";

export function PageHeader() {
	return (
		<header className="flex h-16 items-center justify-between border-b bg-card px-5 md:hidden">
			<Link to="/" className="flex items-center gap-2 font-bold">
				<span className="grid size-8 place-items-center">
					<img src="/my-feed-mascot.png" alt="" className="size-8 object-contain" />
				</span>
				My feed
			</Link>
			<Link
				to="/settings"
				className={buttonVariants({ variant: "ghost", size: "icon" })}
				aria-label="Settings"
			>
				<Settings />
			</Link>
		</header>
	);
}
