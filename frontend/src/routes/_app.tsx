import { createFileRoute, Outlet } from "@tanstack/react-router";
import { DesktopSidebar } from "#/components/layout/DesktopSidebar";
import { MobileNavigation } from "#/components/layout/MobileNavigation";
import { PageHeader } from "#/components/layout/PageHeader";
import { FeedProvider } from "#/features/feed/FeedState";

export const Route = createFileRoute("/_app")({ component: AppLayout });
function AppLayout() {
	return (
		<FeedProvider>
			<div className="min-h-screen bg-background text-foreground md:grid md:grid-cols-[260px_minmax(0,1fr)]">
				<DesktopSidebar />
				<main className="min-w-0 pb-[calc(6rem+env(safe-area-inset-bottom,0px))] md:pb-0">
					<PageHeader />
					<div className="mx-auto max-w-6xl px-5 py-7">
						<Outlet />
					</div>
				</main>
				<MobileNavigation />
			</div>
		</FeedProvider>
	);
}
