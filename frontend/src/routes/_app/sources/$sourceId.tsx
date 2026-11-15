import { createFileRoute } from "@tanstack/react-router";
import { FeedPage } from "#/features/feed/FeedPage";
export const Route = createFileRoute("/_app/sources/$sourceId")({
	component: Source,
});
function Source() {
	const { sourceId } = Route.useParams();
	return <FeedPage key={sourceId} sourceId={sourceId} />;
}
