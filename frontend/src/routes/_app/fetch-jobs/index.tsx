import { createFileRoute } from "@tanstack/react-router";
import { PageContainer } from "#/components/layout/PageContainer";
import { FetchJobsPage } from "#/features/fetch-jobs/FetchJobsPage";

export const Route = createFileRoute("/_app/fetch-jobs/")({
	component: () => (
		<PageContainer>
			<FetchJobsPage />
		</PageContainer>
	),
});
