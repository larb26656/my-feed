import { createFileRoute } from "@tanstack/react-router";
import { FetchJobsPage } from "#/features/fetch-jobs/FetchJobsPage";

export const Route = createFileRoute("/_app/fetch-jobs/")({
	component: FetchJobsPage,
});
