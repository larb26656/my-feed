import { EmptyState, type EmptyStateProps } from "./EmptyState";

export type ErrorStateProps = Omit<
	EmptyStateProps,
	"role" | "description" | "title"
> & {
	error: Error | string;
	message?: string;
	title?: string;
};

export function ErrorState({
	error,
	message,
	title = "An error occurred",
	illustration = "error",
	...props
}: ErrorStateProps) {
	const description =
		message ?? (typeof error === "string" ? error : error.message);

	return (
		<EmptyState
			{...props}
			description={description || "This action is unavailable right now"}
			title={title}
			illustration={illustration}
			role="alert"
		/>
	);
}
