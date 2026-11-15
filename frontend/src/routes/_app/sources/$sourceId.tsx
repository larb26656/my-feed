import { createFileRoute, notFound } from "@tanstack/react-router";
import { entries } from "#/features/feed/data";
import { FeedPage } from "#/features/feed/FeedPage";
export const Route = createFileRoute("/_app/sources/$sourceId")({
	beforeLoad: ({ params }) => {
		if (!entries.some((entry) => entry.sourceId === params.sourceId))
			throw notFound({ routeId: "__root__" });
	},
	component: Source,
});
function Source() {
	const { sourceId } = Route.useParams();
	return <FeedPage key={sourceId} sourceId={sourceId} />;
}
