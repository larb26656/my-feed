import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { DesktopSidebar } from "#/components/layout/DesktopSidebar";
import { MobileNavigation } from "#/components/layout/MobileNavigation";
import { PageHeader } from "#/components/layout/PageHeader";
import { FeedProvider } from "#/features/feed/FeedState";
import { refreshSession } from "#/lib/auth/session";
import { getAccessToken, getRefreshToken } from "#/lib/auth/tokens";

export const Route = createFileRoute("/_app")({
	beforeLoad: async () => {
		if (getAccessToken()) {
			return;
		}
		if (getRefreshToken() && (await refreshSession())) {
			return;
		}
		throw redirect({ to: "/login" });
	},
	component: AppLayout,
});

function AppLayout() {
	return (
		<FeedProvider>
			<div className="min-h-screen bg-background text-foreground md:grid md:grid-cols-[260px_minmax(0,1fr)]">
				<DesktopSidebar />
				<main className="min-w-0 bg-muted/50 pb-[calc(6rem+env(safe-area-inset-bottom,0px))] md:pb-0 dark:bg-background">
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
