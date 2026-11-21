import { zodResolver } from "@hookform/resolvers/zod";
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import { FormField } from "#/components/ui/form-field";
import { login, refreshSession } from "#/lib/auth/session";
import { getAccessToken, getRefreshToken } from "#/lib/auth/tokens";

export const Route = createFileRoute("/login")({
	beforeLoad: async () => {
		if (getAccessToken()) {
			throw redirect({ to: "/" });
		}
		if (getRefreshToken() && (await refreshSession())) {
			throw redirect({ to: "/" });
		}
	},
	component: LoginPage,
});

const loginSchema = z.object({
	username: z.string().trim().min(1, "Please enter a username"),
	password: z.string().min(1, "Please enter a password"),
});

type LoginForm = z.infer<typeof loginSchema>;

function LoginPage() {
	const navigate = useNavigate();
	const [submitError, setSubmitError] = useState<string | null>(null);
	const form = useForm<LoginForm>({
		resolver: zodResolver(loginSchema),
		defaultValues: { username: "", password: "" },
	});

	const submit = async (values: LoginForm) => {
		setSubmitError(null);
		try {
			await login(values.username, values.password);
			await navigate({ to: "/" });
		} catch (error) {
			setSubmitError(
				error instanceof Error ? error.message : "Could not sign in",
			);
		}
	};

	return (
		<main className="flex min-h-screen items-center justify-center bg-background px-5 text-foreground">
			<section className="w-full max-w-sm space-y-6">
				<div className="space-y-2">
					<p className="text-sm font-medium tracking-wide text-muted-foreground uppercase">
						My feed
					</p>
					<h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
					<p className="text-sm text-muted-foreground">
						Use the credentials configured for this instance
					</p>
				</div>
				<form
					className="space-y-4"
					onSubmit={form.handleSubmit(submit)}
					noValidate
				>
					<FormField
						id="username"
						label="Username"
						error={form.formState.errors.username?.message}
						registration={form.register("username")}
					/>
					<FormField
						id="password"
						label="Password"
						type="password"
						error={form.formState.errors.password?.message}
						registration={form.register("password")}
					/>
					{submitError ? (
						<p className="text-sm text-destructive" role="alert">
							{submitError}
						</p>
					) : null}
					<Button
						type="submit"
						className="w-full"
						disabled={form.formState.isSubmitting}
					>
						{form.formState.isSubmitting ? "Signing in…" : "Sign in"}
					</Button>
				</form>
			</section>
		</main>
	);
}
