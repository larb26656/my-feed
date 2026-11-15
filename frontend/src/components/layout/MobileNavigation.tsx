import { Link } from "@tanstack/react-router";
import { Bookmark, Layers3, Rss } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";
import { useFeedState } from "#/features/feed/FeedState";

export function MobileNavigation() {
	const className = buttonVariants({
		variant: "ghost",
		className: "h-full flex-col gap-1 text-xs aria-[current=page]:text-primary",
	});
	return (
		<nav
			className="fixed inset-x-0 bottom-0 z-10 flex h-16 justify-around border-t bg-card md:hidden"
			aria-label="Main navigation"
		>
			<Link to="/" activeOptions={{ exact: true }} className={className}>
				<Layers3 />
				My Feed
			</Link>
			<Link to="/saved" className={className}>
				<Bookmark />
				Saved
			</Link>
			<Link to="/sources" className={className}>
				<Rss />
				Sources
			</Link>
		</nav>
	);
}
