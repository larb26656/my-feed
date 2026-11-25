import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";
import { useEntries } from "#/hooks/queries";
import { FeedEntries } from "./FeedEntries";
import { FeedSort } from "./FeedSort";
import { useFeedSort } from "./useFeedSort";

export function FeedPage() {
	const sort = useFeedSort("publishedAtSort");
	const query = useEntries({
		sortField: sort.sortField,
		sortDirection: sort.sortDirection,
	});
	return (
		<section className="mx-auto max-w-6xl md:px-5 md:py-7">
			<h1 className="sr-only">My Feed</h1>
			<div className="mb-4 flex items-center justify-end gap-3 px-4 py-2 md:px-0">
				<FeedSort {...sort} />
				<div className="h-6 border-l border-border/70" aria-hidden="true" />
				<Link
					to="/record-link"
					aria-label="Save Link"
					className={buttonVariants({ variant: "ghost", size: "icon-sm" })}
				>
					<Plus />
				</Link>
			</div>
			<FeedEntries query={query} />
		</section>
	);
}
