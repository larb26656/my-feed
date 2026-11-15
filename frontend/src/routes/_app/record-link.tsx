import { createFileRoute } from "@tanstack/react-router";
import { RecordLinkPage } from "#/features/record-link/RecordLinkPage";

export const Route = createFileRoute("/_app/record-link")({
	component: RecordLinkPage,
});
