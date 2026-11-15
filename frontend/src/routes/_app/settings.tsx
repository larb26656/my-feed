import { createFileRoute } from "@tanstack/react-router";
import { NoDataState } from "#/components/NoDataState";
export const Route = createFileRoute("/_app/settings")({ component: Settings });
function Settings() {
	return (
		<section>
			<h1 className="text-2xl font-bold">Settings</h1>
			<NoDataState
				illustration="search"
				title="ยังไม่มีการตั้งค่า"
				description="การตั้งค่าที่ปรับได้จะปรากฏที่นี่เมื่อพร้อมใช้งาน"
			/>
		</section>
	);
}
