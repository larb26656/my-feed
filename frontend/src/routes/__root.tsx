import { Link, Outlet, createRootRoute } from "@tanstack/react-router";

import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { TanStackDevtools } from "@tanstack/react-devtools";

import "../styles.css";

export const Route = createRootRoute({
	component: RootComponent,
	notFoundComponent: NotFound,
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
		<main className="grid min-h-screen place-items-center bg-background px-5 text-center text-foreground">
			<div>
				<p className="text-sm text-muted-foreground">404</p>
				<h1 className="mt-2 text-2xl font-bold">ไม่พบหน้านี้</h1>
				<Link to="/" className="mt-6 inline-block text-primary underline">
					กลับไปที่ฟีด
				</Link>
			</div>
		</main>
	);
}
