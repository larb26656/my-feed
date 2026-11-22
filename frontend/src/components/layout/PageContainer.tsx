import type { ReactNode } from "react";

export function PageContainer({ children }: { children: ReactNode }) {
	return (
		<div className="mx-auto max-w-6xl px-4 py-4 md:px-5 md:py-7">
			{children}
		</div>
	);
}
