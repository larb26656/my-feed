import { zodResolver } from "@hookform/resolvers/zod";
import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
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
				<form onSubmit={form.handleSubmit(submit)} noValidate>
					<FieldGroup>
						<Controller
							name="username"
							control={form.control}
							render={({ field, fieldState }) => (
								<Field data-invalid={fieldState.invalid}>
									<FieldLabel htmlFor="username">Username</FieldLabel>
									<Input
										{...field}
										id="username"
										aria-invalid={fieldState.invalid}
										aria-describedby={
											fieldState.invalid ? "username-error" : undefined
										}
									/>
									{fieldState.invalid && (
										<FieldError
											id="username-error"
											errors={[fieldState.error]}
										/>
									)}
								</Field>
							)}
						/>
						<Controller
							name="password"
							control={form.control}
							render={({ field, fieldState }) => (
								<Field data-invalid={fieldState.invalid}>
									<FieldLabel htmlFor="password">Password</FieldLabel>
									<Input
										{...field}
										id="password"
										type="password"
										aria-invalid={fieldState.invalid}
										aria-describedby={
											fieldState.invalid ? "password-error" : undefined
										}
									/>
									{fieldState.invalid && (
										<FieldError
											id="password-error"
											errors={[fieldState.error]}
										/>
									)}
								</Field>
							)}
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
					</FieldGroup>
				</form>
			</section>
		</main>
	);
}
