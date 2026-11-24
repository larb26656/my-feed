import { Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "#/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "#/components/ui/tabs";
import { toastManager } from "#/components/ui/toaster";
import type { Source } from "#/features/feed/types";
import { SourceTableView } from "#/features/sources/desktop/SourceTableView";
import { SourceListView } from "#/features/sources/mobile/SourceListView";
import { ProfileImageDialog } from "#/features/sources/ProfileImageDialog";
import type { SourceActionHandlers } from "#/features/sources/SourceActions";
import { SourceDialog } from "#/features/sources/SourceDialog";
import { SourceToolbar } from "#/features/sources/SourceToolbar";
import type { SourceFilters, SourceSort } from "#/features/sources/source.api";
import { useIsMobile } from "#/features/sources/useIsMobile";
import { useSetSourceActive, useSources } from "#/hooks/queries";

type SourcesTab = "active" | "inactive";

function SourcesHeader({
	isMobile,
	onCreate,
}: {
	isMobile: boolean;
	onCreate: () => void;
}) {
	if (isMobile) {
		return (
			<div className="flex items-center justify-between gap-3">
				<h1 className="text-xl font-semibold tracking-tight">Sources</h1>
				<Button onClick={onCreate}>
					<Plus data-icon="inline-start" />
					Add Source
				</Button>
			</div>
		);
	}

	return (
		<div className="flex flex-wrap items-start justify-between gap-4">
			<div>
				<h1 className="sr-only">Manage Sources</h1>
				<p className="mt-2 text-sm text-muted-foreground">
					Add, edit, and activate the feeds you follow
				</p>
			</div>
			<Button onClick={onCreate}>Add Source</Button>
		</div>
	);
}

export function SourcesPage({ focusSourceId }: { focusSourceId?: string }) {
	const isMobile = useIsMobile();
	const [tab, setTab] = useState<SourcesTab>("active");
	const [appliedFilters, setAppliedFilters] = useState<SourceFilters>({
		name: "",
		type: "",
	});
	const [sort, setSort] = useState<SourceSort>({
		sortBy: "name",
		sortDirection: "asc",
	});
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
		setSearchRevision((current) => current + 1);
	}, [focusSourceId, focused]);
	const [imageSource, setImageSource] = useState<Source | null>(null);
	const [editing, setEditing] = useState<Source | null>(null);
	const [showForm, setShowForm] = useState(false);
	const activationMutation = useSetSourceActive();

	const applyFilters = (filters: SourceFilters) => {
		setAppliedFilters((current) =>
			current.name === filters.name && current.type === filters.type
				? current
				: filters,
		);
	};
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
	};
	const actions: SourceActionHandlers = {
		activationPending: activationMutation.isPending,
		activationSourceId: activationMutation.variables?.id,
		onEdit: openEdit,
		onImage: setImageSource,
		onToggleActive: (item) => {
			activationMutation.reset();
			activationMutation.mutate(
				{ id: item.id, active: !item.active },
				{ onSuccess: handleActivationSuccess },
			);
		},
	};
	const active = tab === "active";

	return (
		<section>
			<SourcesHeader isMobile={isMobile} onCreate={openCreate} />

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

			<Tabs
				value={tab}
				onValueChange={onTabChange}
				className={isMobile ? "mt-5" : "mt-8"}
			>
				<TabsList variant="line">
					<TabsTrigger value="active">Active</TabsTrigger>
					<TabsTrigger value="inactive">Inactive</TabsTrigger>
				</TabsList>
				<SourceToolbar
					key={`${searchRevision}-${isMobile ? "mobile" : "desktop"}`}
					variant={isMobile ? "mobile" : "desktop"}
					initialName={appliedFilters.name}
					initialType={appliedFilters.type || "all"}
					sort={sort}
					onSearch={applyFilters}
					onSortChange={setSort}
				/>
				{isMobile ? (
					<SourceListView
						active={active}
						filters={appliedFilters}
						sort={sort}
						actions={actions}
					/>
				) : (
					<SourceTableView
						active={active}
						filters={appliedFilters}
						sort={sort}
						onSortChange={setSort}
						actions={actions}
					/>
				)}
			</Tabs>
		</section>
	);
}
