import { Camera, Ellipsis, Pencil, Power } from "lucide-react";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import type { Source } from "#/features/feed/types";

export type SourceActionHandlers = {
	activationPending: boolean;
	activationSourceId?: string;
	onEdit: (source: Source) => void;
	onImage: (source: Source) => void;
	onToggleActive: (source: Source) => void;
};

type SourceActionsProps = SourceActionHandlers & {
	source: Source;
	appearance?: "outline" | "ghost";
};

const menuItemClass =
	"flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted disabled:opacity-50";

export function SourceActions({
	source,
	activationPending,
	activationSourceId,
	onEdit,
	onImage,
	onToggleActive,
	appearance = "outline",
}: SourceActionsProps) {
	const [menuOpen, setMenuOpen] = useState(false);

	if (source.type === "manual") return null;

	const pendingHere = activationPending && activationSourceId === source.id;
	const activationLabel = pendingHere
		? source.active
			? "Deactivating..."
			: "Activating..."
		: source.active
			? "Deactivate source"
			: "Activate source";

	return (
		<div className="relative flex justify-end">
			<Button
				variant={appearance}
				size="icon"
				aria-label={`Actions for ${source.name}`}
				aria-haspopup="menu"
				aria-expanded={menuOpen}
				onClick={() => setMenuOpen((open) => !open)}
			>
				<Ellipsis size={18} />
			</Button>
			{menuOpen && (
				<div
					role="menu"
					aria-label={`${source.name} actions`}
					className="absolute right-0 top-full z-30 mt-2 min-w-52 rounded-xl border bg-popover p-1 shadow-lg"
				>
					<button
						type="button"
						role="menuitem"
						className={menuItemClass}
						onClick={() => {
							setMenuOpen(false);
							onEdit(source);
						}}
					>
						<Pencil size={14} />
						Edit source
					</button>
					<button
						type="button"
						role="menuitem"
						className={menuItemClass}
						onClick={() => {
							setMenuOpen(false);
							onImage(source);
						}}
					>
						<Camera size={14} aria-hidden="true" />
						Change profile image
					</button>
					<button
						type="button"
						role="menuitem"
						disabled={activationPending}
						className={menuItemClass}
						onClick={() => {
							setMenuOpen(false);
							onToggleActive(source);
						}}
					>
						<Power size={14} />
						{activationLabel}
					</button>
				</div>
			)}
		</div>
	);
}
