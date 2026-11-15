import { Link } from "@tanstack/react-router";
import { Rss, Settings } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";

export function PageHeader() {
	return (
		<header className="flex h-16 items-center justify-between border-b bg-card px-5 md:hidden">
			<Link to="/" className="flex items-center gap-2 font-bold">
				<span className="grid size-8 place-items-center rounded-lg bg-primary text-primary-foreground">
					<Rss size={17} />
				</span>
				quick.feed
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
