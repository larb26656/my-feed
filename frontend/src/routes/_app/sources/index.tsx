import { createFileRoute } from "@tanstack/react-router";
import { PageContainer } from "#/components/layout/PageContainer";
import { SourcesPage } from "#/features/sources/SourcesPage";

type SourcesSearchParams = {
	sourceId?: string;
};

export const Route = createFileRoute("/_app/sources/")({
	validateSearch: (search: Record<string, unknown>): SourcesSearchParams => ({
		sourceId:
			typeof search.sourceId === "string" && search.sourceId.length > 0
				? search.sourceId
				: undefined,
	}),
	component: SourcesRoute,
});

function SourcesRoute() {
	const { sourceId } = Route.useSearch();
	return (
		<PageContainer>
			<SourcesPage focusSourceId={sourceId} />
		</PageContainer>
	);
}
