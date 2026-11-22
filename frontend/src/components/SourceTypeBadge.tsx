import { Layers3, Rss } from "lucide-react";
import { Badge } from "#/components/ui/badge";

const types: Record<
	string,
	{ label: string; icon: typeof Rss; iconClassName?: string }
> = {
	rss: { label: "RSS", icon: Rss },
	manual: { label: "Manual", icon: Layers3, iconClassName: "text-blue-600" },
};

export function SourceTypeBadge({
	type,
	variant = "outline",
}: {
	type: string;
	variant?: "outline" | "inline";
}) {
	const sourceType = types[type];
	const Icon = sourceType?.icon;

	return (
		<Badge
			variant="outline"
			className={
				variant === "inline"
					? "h-auto border-0 p-0 font-normal text-muted-foreground"
					: undefined
			}
		>
			{type === "youtube" && (
				<svg aria-hidden="true" viewBox="0 0 24 24">
					<rect x="2" y="4" width="20" height="16" rx="4" fill="#ff0000" />
					<path d="M10 8.5v7l6-3.5z" fill="white" />
				</svg>
			)}
			{Icon && <Icon aria-hidden="true" className={sourceType.iconClassName} />}
			{type === "youtube" ? "YouTube" : (sourceType?.label ?? type)}
		</Badge>
	);
}
