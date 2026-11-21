import { Ellipsis, Pencil, Power } from "lucide-react";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import type { Source } from "#/features/feed/types";

type SourcesItemProps = {
	source: Source;
	activationPending: boolean;
	activationSourceId?: string;
	onEdit: (source: Source) => void;
	onToggleActive: (source: Source) => void;
};

export function SourcesItem({
	source,
	activationPending,
	activationSourceId,
	onEdit,
	onToggleActive,
}: SourcesItemProps) {
	const [menuOpen, setMenuOpen] = useState(false);

	if (source.type === "manual") return null;

	return (
		<div className="relative flex justify-end">
			<Button
				variant="outline"
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
					className="absolute right-0 top-full z-20 mt-2 min-w-52 rounded-xl border bg-popover p-1 shadow-lg"
				>
					<button
						type="button"
						role="menuitem"
						className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted"
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
						disabled={activationPending}
						className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted disabled:opacity-50"
						onClick={() => {
							setMenuOpen(false);
							onToggleActive(source);
						}}
					>
						<Power size={14} />
						{activationPending && activationSourceId === source.id
							? source.active
								? "Deactivating..."
								: "Activating..."
							: source.active
								? "Deactivate source"
								: "Activate source"}
					</button>
				</div>
			)}
		</div>
	);
}
