import { EmptyState, type EmptyStateProps } from "./EmptyState";

export type NoDataStateProps = Omit<EmptyStateProps, "role">;

export function NoDataState(props: NoDataStateProps) {
	return <EmptyState {...props} role="status" />;
}
