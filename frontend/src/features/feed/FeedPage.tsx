import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { NoDataState } from "#/components/NoDataState";
import { PullToRefresh } from "#/components/PullToRefresh";
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
		<PullToRefresh onRefresh={() => query.refetch()}>
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
				<FeedEntries
					query={query}
					emptyState={
						<NoDataState
							illustration="feed"
							title="Nothing to read yet"
							description="Add sources to follow or save links you want to read here"
							action={
								<Link to="/record-link" className={buttonVariants()}>
									Save Link
								</Link>
							}
						/>
					}
				/>
			</section>
		</PullToRefresh>
	);
}
