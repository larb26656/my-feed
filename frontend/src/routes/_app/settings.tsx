import { createFileRoute } from "@tanstack/react-router";
export const Route = createFileRoute("/_app/settings")({ component: Settings });
function Settings() {
	return (
		<section>
			<h1 className="text-2xl font-bold">Settings</h1>
			<p className="mt-3 text-sm text-muted-foreground">
				ยังไม่มีการตั้งค่าที่ปรับได้ในขณะนี้
			</p>
		</section>
	);
}
