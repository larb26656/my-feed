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
				imageAlt="มาสคอตกำลังสงสัย"
				title="ไม่พบหน้านี้"
				description="หน้าที่คุณกำลังหาอาจถูกย้ายหรือลบไปแล้ว"
				action={
					<Link to="/" className="text-primary underline underline-offset-4">
						กลับไปที่ฟีด
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
				title="เกิดข้อผิดพลาดที่ไม่คาดคิด"
				message="ระบบไม่สามารถแสดงหน้านี้ได้ ลองอีกครั้งหรือกลับไปที่ฟีด"
				action={
					<div className="flex flex-wrap justify-center gap-3">
						<Button onClick={reset}>ลองอีกครั้ง</Button>
						<Button variant="outline" render={<Link to="/" />}>
							กลับไปที่ฟีด
						</Button>
					</div>
				}
			/>
		</main>
	);
}
