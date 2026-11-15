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
	title = "เกิดข้อผิดพลาด",
	illustration = "error",
	...props
}: ErrorStateProps) {
	const description =
		message ?? (typeof error === "string" ? error : error.message);

	return (
		<EmptyState
			{...props}
			description={description || "ไม่สามารถดำเนินการได้ในขณะนี้"}
			title={title}
			illustration={illustration}
			role="alert"
		/>
	);
}
