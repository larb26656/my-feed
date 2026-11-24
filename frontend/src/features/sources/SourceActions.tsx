import { Camera, Ellipsis, Pencil, Power } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "#/components/ui/dropdown-menu";
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

export function SourceActions({
	source,
	activationPending,
	activationSourceId,
	onEdit,
	onImage,
	onToggleActive,
	appearance = "outline",
}: SourceActionsProps) {
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
		<DropdownMenu>
			<DropdownMenuTrigger
				render={
					<Button
						variant={appearance}
						size="icon"
						aria-label={`Actions for ${source.name}`}
					/>
				}
			>
				<Ellipsis />
			</DropdownMenuTrigger>
			<DropdownMenuContent
				align="end"
				className="min-w-52"
				aria-label={`${source.name} actions`}
			>
				<DropdownMenuGroup>
					<DropdownMenuItem onClick={() => onEdit(source)}>
						<Pencil />
						Edit source
					</DropdownMenuItem>
					<DropdownMenuItem onClick={() => onImage(source)}>
						<Camera />
						Change profile image
					</DropdownMenuItem>
					<DropdownMenuItem
						disabled={activationPending}
						onClick={() => onToggleActive(source)}
					>
						<Power />
						{activationLabel}
					</DropdownMenuItem>
				</DropdownMenuGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
