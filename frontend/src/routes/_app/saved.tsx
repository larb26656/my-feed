import { createFileRoute } from "@tanstack/react-router";
import { SavedPage } from "#/features/feed/SavedPage";
export const Route = createFileRoute("/_app/saved")({
	component: SavedPage,
});
