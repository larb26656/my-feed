import { Link } from "@tanstack/react-router";
import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import type { Source } from "#/features/feed/types";
import {
	type SourceActionHandlers,
	SourceActions,
} from "#/features/sources/SourceActions";

export function SourceCard({
	source,
	actions,
}: {
	source: Source;
	actions: SourceActionHandlers;
}) {
	return (
		<li className="flex items-center gap-3 rounded-xl border bg-card px-3 py-3">
			<SourceAvatar
				name={source.name}
				imageId={source.profileImageId}
				sourceId={source.id}
				size={40}
			/>
			<div className="min-w-0 flex-1">
				<Link
					to="/sources/$sourceId"
					params={{ sourceId: source.id }}
					className="block truncate font-medium hover:underline"
				>
					{source.name}
				</Link>
				<div className="mt-1">
					<SourceTypeBadge type={source.type} />
				</div>
			</div>
			<SourceActions source={source} appearance="ghost" {...actions} />
		</li>
	);
}
