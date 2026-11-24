import { Badge } from "#/components/ui/badge";
import { getSourceTypePresentation } from "#/lib/source-type-presentation";

export function SourceTypeBadge({
	type,
	variant = "outline",
}: {
	type: string;
	variant?: "outline" | "inline";
}) {
	const { label, icon: Icon } = getSourceTypePresentation(type);

	return (
		<Badge
			variant="outline"
			className={
				variant === "inline"
					? "h-auto border-0 p-0 font-normal text-muted-foreground"
					: undefined
			}
		>
			<Icon
				aria-hidden="true"
				className={type === "manual" ? "text-blue-600" : undefined}
			/>
			{label}
		</Badge>
	);
}
