import { createFileRoute, Outlet } from "@tanstack/react-router";
import { DesktopSidebar } from "#/components/layout/DesktopSidebar";
import { MobileNavigation } from "#/components/layout/MobileNavigation";
import { PageHeader } from "#/components/layout/PageHeader";
import { FeedProvider } from "#/features/feed/FeedState";

export const Route = createFileRoute("/_app")({ component: AppLayout });
function AppLayout() {
	return (
		<FeedProvider>
			<div className="min-h-screen bg-background text-foreground md:grid md:grid-cols-[220px_minmax(0,1fr)]">
				<DesktopSidebar />
				<main className="min-w-0 pb-20 md:pb-0">
					<PageHeader />
					<div className="mx-auto max-w-[910px] px-5 pb-16 pt-9 md:px-10 md:pt-12">
						<Outlet />
					</div>
				</main>
				<MobileNavigation />
			</div>
		</FeedProvider>
	);
}
