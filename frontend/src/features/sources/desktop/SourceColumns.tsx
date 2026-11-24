import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import type { Source } from "#/features/feed/types";
import {
	type SourceActionHandlers,
	SourceActions,
} from "#/features/sources/SourceActions";
import { formatDate } from "#/lib/date";

function sourceDetail(source: Source) {
	const value =
		source.type === "youtube"
			? source.config?.channelId
			: source.type === "medium"
				? source.config?.identifier
				: source.type === "bluesky"
					? source.config?.did
					: source.config?.url;
	if (typeof value === "string") return value;
	return "—";
}

export function sourceColumns(
	actions: SourceActionHandlers,
): ColumnDef<Source, unknown>[] {
	return [
		{
			accessorKey: "name",
			header: "Name",
			cell: ({ row }) => {
				const source = row.original;
				return (
					<div className="flex min-w-0 items-center gap-3">
						<SourceAvatar
							name={source.name}
							imageId={source.profileImageId}
							sourceId={source.id}
							size={32}
						/>
						<Link
							to="/sources/$sourceId"
							params={{ sourceId: source.id }}
							className="truncate font-medium hover:underline"
						>
							{source.name}
						</Link>
					</div>
				);
			},
		},
		{
			accessorKey: "type",
			header: "Type",
			cell: ({ row }) => <SourceTypeBadge type={row.original.type} />,
		},
		{
			id: "detail",
			header: "Feed / Channel",
			enableSorting: false,
			cell: ({ row }) => (
				<span className="block max-w-72 truncate text-muted-foreground">
					{sourceDetail(row.original)}
				</span>
			),
		},
		{
			accessorKey: "createdAt",
			header: "Date added",
			cell: ({ row }) =>
				row.original.createdAt ? (
					<time dateTime={row.original.createdAt}>
						{formatDate(row.original.createdAt)}
					</time>
				) : (
					"—"
				),
		},
		{
			id: "actions",
			enableSorting: false,
			header: () => <span className="sr-only">Actions</span>,
			cell: ({ row }) => <SourceActions source={row.original} {...actions} />,
		},
	];
}
