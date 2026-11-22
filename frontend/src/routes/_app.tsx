import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { AppSidebar } from "#/components/layout/AppSidebar";
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
	const [collapsed, setCollapsed] = useState(
		() => localStorage.getItem("sidebar-collapsed") === "true",
	);
	const toggleSidebar = () => {
		setCollapsed((previous) => {
			localStorage.setItem("sidebar-collapsed", String(!previous));
			return !previous;
		});
	};
	return (
		<FeedProvider>
			<div
				className={`min-h-screen bg-background text-foreground motion-safe:transition-[grid-template-columns] motion-safe:duration-300 motion-safe:ease-out md:grid ${collapsed ? "md:grid-cols-[72px_minmax(0,1fr)]" : "md:grid-cols-[260px_minmax(0,1fr)]"}`}
			>
				<div className="sticky top-0 hidden h-screen border-r md:block">
					<AppSidebar collapsed={collapsed} onToggle={toggleSidebar} />
				</div>
				<main className="min-w-0 bg-muted/50 pb-[calc(6rem+env(safe-area-inset-bottom,0px))] md:pb-0 dark:bg-background">
					<PageHeader />
					<Outlet />
				</main>
				<MobileNavigation />
			</div>
		</FeedProvider>
	);
}
