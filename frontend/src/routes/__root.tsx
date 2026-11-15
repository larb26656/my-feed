import { Link, Outlet, createRootRoute } from "@tanstack/react-router";

import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { EmptyState } from "#/components/EmptyState";
import { ErrorState } from "#/components/ErrorState";
import { Button } from "#/components/ui/button";

import "../styles.css";

export const Route = createRootRoute({
	component: RootComponent,
	notFoundComponent: NotFound,
	errorComponent: GlobalError,
});

function RootComponent() {
	return (
		<>
			<Outlet />
			<TanStackDevtools
				config={{
					position: "bottom-right",
				}}
				plugins={[
					{
						name: "TanStack Router",
						render: <TanStackRouterDevtoolsPanel />,
					},
				]}
			/>
		</>
	);
}

function NotFound() {
	return (
		<main className="bg-background text-foreground">
			<EmptyState
				size="screen"
				illustration="/empty-states/confused.png"
				imageAlt="Curious mascot"
				title="Page not found"
				description="The page you are looking for may have been moved or deleted"
				action={
					<Link to="/" className="text-primary underline underline-offset-4">
						Back to feed
					</Link>
				}
			/>
		</main>
	);
}

function GlobalError({ error, reset }: { error: Error; reset: () => void }) {
	return (
		<main className="bg-background text-foreground">
			<ErrorState
				size="screen"
				error={error}
				title="Unexpected error"
				message="We could not display this page. Try again or return to the feed"
				action={
					<div className="flex flex-wrap justify-center gap-3">
						<Button onClick={reset}>Try again</Button>
						<Button variant="outline" render={<Link to="/" />}>
							Back to feed
						</Button>
					</div>
				}
			/>
		</main>
	);
}
