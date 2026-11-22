import { createFileRoute } from "@tanstack/react-router";
import { PageContainer } from "#/components/layout/PageContainer";
import { RecordLinkPage } from "#/features/record-link/RecordLinkPage";

export const Route = createFileRoute("/_app/record-link")({
	component: () => (
		<PageContainer>
			<RecordLinkPage />
		</PageContainer>
	),
});
