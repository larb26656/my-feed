import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { PageContainer } from "#/components/layout/PageContainer";
import { Button } from "#/components/ui/button";
import { logout } from "#/lib/auth/session";

export const Route = createFileRoute("/_app/settings")({ component: Settings });

function Settings() {
	const navigate = useNavigate();
	const [busy, setBusy] = useState(false);

	const signOut = async () => {
		setBusy(true);
		try {
			await logout();
			await navigate({ to: "/login" });
		} finally {
			setBusy(false);
		}
	};

	return (
		<PageContainer>
			<section className="space-y-4">
				<h1 className="text-xl font-semibold tracking-tight">Settings</h1>
				<p className="text-sm text-muted-foreground">
					Sign out of this browser Session. Other devices stay signed in.
				</p>
				<Button variant="outline" onClick={signOut} disabled={busy}>
					{busy ? "Signing out…" : "Sign out"}
				</Button>
			</section>
		</PageContainer>
	);
}
