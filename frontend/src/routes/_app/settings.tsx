import { createFileRoute } from "@tanstack/react-router";
import { NoDataState } from "#/components/NoDataState";
export const Route = createFileRoute("/_app/settings")({ component: Settings });
function Settings() {
	return (
		<section>
			<h1 className="text-2xl font-bold">Settings</h1>
			<NoDataState
				illustration="search"
				title="No settings yet"
				description="Available settings will appear here when they are ready"
			/>
		</section>
	);
}
