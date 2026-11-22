import { Link } from "@tanstack/react-router";
import type { ColumnDef, SortingState } from "@tanstack/react-table";
import { useEffect, useRef, useState } from "react";
import { DataTable } from "#/components/data-table/DataTable";
import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import { Button } from "#/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "#/components/ui/tabs";
import { toastManager } from "#/components/ui/toaster";
import type { Source } from "#/features/feed/types";
import { ProfileImageDialog } from "#/features/sources/ProfileImageDialog";
import { SourceDialog } from "#/features/sources/SourceDialog";
import { SourcesItem } from "#/features/sources/SourcesItem";
import { SourcesSearch } from "#/features/sources/SourcesSearch";
import {
	type SourcesPageSize,
	useSetSourceActive,
	useSources,
	useSourcesPage,
} from "#/hooks/queries";
import { formatDate } from "#/lib/date";

type SourcesTab = "active" | "inactive";

const pageSizeOptions = [
	10, 20, 50,
] as const satisfies readonly SourcesPageSize[];

function sourceDetail(source: Source) {
	if (source.type === "youtube") return source.youtubeChannelId ?? "—";
	if (source.type === "rss") return source.rssUrl ?? "—";
	return "—";
}

export function SourcesPage({ focusSourceId }: { focusSourceId?: string }) {
	const [tab, setTab] = useState<SourcesTab>("active");
	const [pageIndex, setPageIndex] = useState(0);
	const [pageSize, setPageSize] = useState<SourcesPageSize>(10);
	const [appliedFilters, setAppliedFilters] = useState({ name: "", type: "" });
	const [searchRevision, setSearchRevision] = useState(0);
	const appliedFocus = useRef<string | null>(null);
	const focusQuery = useSources(Boolean(focusSourceId));
	const focused = focusSourceId
		? focusQuery.data?.find((item) => item.id === focusSourceId)
		: undefined;
	useEffect(() => {
		if (!focusSourceId || !focused || appliedFocus.current === focusSourceId) {
			return;
		}
		appliedFocus.current = focusSourceId;
		setTab(focused.active ? "active" : "inactive");
		setAppliedFilters({ name: focused.name, type: focused.type });
		setPageIndex(0);
		setSearchRevision((current) => current + 1);
	}, [focusSourceId, focused]);
	const [sorting, setSorting] = useState<SortingState>([
		{ id: "name", desc: false },
	]);
	const [imageSource, setImageSource] = useState<Source | null>(null);
	const [editing, setEditing] = useState<Source | null>(null);
	const [showForm, setShowForm] = useState(false);
	const activationMutation = useSetSourceActive();
	const sourcesQuery = useSourcesPage(
		tab === "active",
		pageIndex + 1,
		pageSize,
		appliedFilters,
		{
			sortBy: sorting[0]?.id ?? "name",
			sortDirection: sorting[0]?.desc ? "desc" : "asc",
		},
	);
	const sources = sourcesQuery.data?.data ?? [];
	const totalPages = Math.max(sourcesQuery.data?.meta.totalPages ?? 1, 1);

	const handleActivationSuccess = (source: Source) => {
		toastManager.add({
			title: source.active ? "Source activated" : "Source deactivated",
			type: "success",
		});
	};
	const openCreate = () => {
		setEditing(null);
		setShowForm(true);
	};
	const openEdit = (source: Source) => {
		setEditing(source);
		setShowForm(true);
	};
	const onTabChange = (next: string | number | null) => {
		if (next !== "active" && next !== "inactive") return;
		setTab(next);
		setPageIndex(0);
	};
	const onPageSizeChange = (next: number) => {
		setPageSize(next as SourcesPageSize);
		setPageIndex(0);
	};

	const columns: ColumnDef<Source, unknown>[] = [
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
			header: () => <span className="sr-only">Actions</span>,
			cell: ({ row }) => (
				<SourcesItem
					source={row.original}
					activationPending={activationMutation.isPending}
					activationSourceId={activationMutation.variables?.id}
					onEdit={openEdit}
					onImage={setImageSource}
					onToggleActive={(item) => {
						activationMutation.reset();
						activationMutation.mutate(
							{ id: item.id, active: !item.active },
							{ onSuccess: handleActivationSuccess },
						);
					}}
				/>
			),
		},
	];

	const tableContent = (
		<DataTable
			columns={columns}
			data={sources}
			isLoading={sourcesQuery.isFetching}
			error={
				sourcesQuery.isError
					? {
							error: sourcesQuery.error ?? "Sources unavailable",
							message: "Could not load sources. Please try again",
							action: (
								<Button onClick={() => void sourcesQuery.refetch()}>
									Try again
								</Button>
							),
						}
					: undefined
			}
			getRowId={(source) => source.id}
			ariaLabel={`${tab} sources`}
			sorting={{
				value: sorting,
				onChange: (updater) => {
					setSorting(updater);
					setPageIndex(0);
				},
			}}
			emptyMessage={
				appliedFilters.name || appliedFilters.type
					? "No sources match your filters"
					: tab === "active"
						? "No active sources"
						: "No inactive sources"
			}
			pagination={{
				mode: "page",
				pageIndex,
				pageCount: totalPages,
				pageSize,
				pageSizeOptions: [...pageSizeOptions],
				isLoading: sourcesQuery.isFetching,
				onPageChange: setPageIndex,
				onPageSizeChange,
			}}
		/>
	);

	return (
		<section>
			<div className="flex flex-wrap items-start justify-between gap-4">
				<div>
					<h1 className="sr-only">Manage Sources</h1>
					<p className="mt-2 text-sm text-muted-foreground">
						Add, edit, and activate the feeds you follow
					</p>
				</div>
				<Button onClick={openCreate}>Add Source</Button>
			</div>

			{imageSource && (
				<ProfileImageDialog
					source={imageSource}
					onClose={() => setImageSource(null)}
				/>
			)}
			{showForm && (
				<SourceDialog editing={editing} onClose={() => setShowForm(false)} />
			)}

			{activationMutation.isError && (
				<p role="alert" className="mt-4 text-sm text-destructive">
					Could not update source status: {activationMutation.error.message}
				</p>
			)}

			<Tabs value={tab} onValueChange={onTabChange} className="mt-8">
				<TabsList variant="line">
					<TabsTrigger value="active">Active</TabsTrigger>
					<TabsTrigger value="inactive">Inactive</TabsTrigger>
				</TabsList>
				<SourcesSearch
					key={searchRevision}
					initialName={appliedFilters.name}
					initialType={appliedFilters.type || "all"}
					onSearch={(filters) => {
						setAppliedFilters(filters);
						setPageIndex(0);
					}}
				/>
				<TabsContent value="active" className="mt-4">
					{tab === "active" && tableContent}
				</TabsContent>
				<TabsContent value="inactive" className="mt-4">
					{tab === "inactive" && tableContent}
				</TabsContent>
			</Tabs>
		</section>
	);
}
