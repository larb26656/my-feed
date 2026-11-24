import { createFileRoute } from "@tanstack/react-router";
import { SourceFeedPage } from "#/features/feed/SourceFeedPage";
export const Route = createFileRoute("/_app/sources/$sourceId")({
	component: Source,
});
function Source() {
	const { sourceId } = Route.useParams();
	return <SourceFeedPage key={sourceId} sourceId={sourceId} />;
}
