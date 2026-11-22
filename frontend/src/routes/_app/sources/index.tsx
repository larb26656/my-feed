import { createFileRoute } from "@tanstack/react-router";
import { PageContainer } from "#/components/layout/PageContainer";
import { SourcesPage } from "#/features/sources/SourcesPage";

export const Route = createFileRoute("/_app/sources/")({
	component: () => (
		<PageContainer>
			<SourcesPage />
		</PageContainer>
	),
});
