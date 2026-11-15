import { cn } from "#/lib/utils";
import type { Entry } from "./data";

export const sourceColors: Record<string, string> = {
	ink: "bg-foreground",
	violet: "bg-violet-500",
	orange: "bg-orange-400",
	sand: "bg-stone-400",
};
export const artColors: Record<string, string> = {
	city: "bg-emerald-100 text-emerald-900",
	creative: "bg-violet-100 text-violet-900",
	tech: "bg-green-100 text-green-900",
	sunday: "bg-amber-100 text-amber-900",
};
export function SourceAvatar({ entry }: { entry: Entry }) {
	return (
		<span
			className={cn(
				"grid size-7 shrink-0 place-items-center rounded-full text-xs font-bold text-white",
				sourceColors[entry.color],
			)}
		>
			{entry.initials}
		</span>
	);
}
