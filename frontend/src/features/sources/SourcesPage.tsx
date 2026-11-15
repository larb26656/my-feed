import { zodResolver } from "@hookform/resolvers/zod";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { Button } from "#/components/ui/button";
import { FormField } from "#/components/ui/form-field";
import { Skeleton } from "#/components/ui/skeleton";
import { toastManager } from "#/components/ui/toaster";
import { useFeedState } from "#/features/feed/FeedState";
import type { Source } from "#/features/feed/types";
import { useSaveSource, useSetSourceActive } from "#/hooks/queries";

const sourceSchema = z.object({
	name: z.string().trim().min(1, "Please enter a source name"),
	rssUrl: z.url("Please enter a valid RSS URL"),
});
type SourceForm = z.infer<typeof sourceSchema>;

export function SourcesPage() {
	const {
		allSources,
		sourcesLoading,
		sourcesError,
		sourcesErrorValue,
		refetchSources,
	} = useFeedState();
	const [editing, setEditing] = useState<Source | null>(null);
	const [showForm, setShowForm] = useState(false);
	const form = useForm<SourceForm>({
		resolver: zodResolver(sourceSchema),
		defaultValues: { name: "", rssUrl: "" },
	});
	const mutation = useSaveSource();
	const activationMutation = useSetSourceActive();
	const handleSaveSuccess = () => {
		toastManager.add({
			title: editing ? "Source updated" : "Source added",
			type: "success",
		});
		setEditing(null);
		setShowForm(false);
		form.reset({ name: "", rssUrl: "" });
	};
	const handleActivationSuccess = (source: Source) => {
		toastManager.add({
			title: source.active ? "Source activated" : "Source deactivated",
			type: "success",
		});
	};
	const openCreate = () => {
		mutation.reset();
		setEditing(null);
		form.reset({ name: "", rssUrl: "" });
		setShowForm(true);
	};
	const openEdit = (source: Source) => {
		mutation.reset();
		setEditing(source);
		form.reset({ name: source.name, rssUrl: source.rssUrl ?? "" });
		setShowForm(true);
	};

	return (
		<section>
			<div className="flex flex-wrap items-start justify-between gap-4">
				<div>
					<h1 className="text-2xl font-bold">Your Sources</h1>
					<p className="mt-2 text-sm text-muted-foreground">
						Choose a source to view its articles
					</p>
				</div>
				<Button onClick={openCreate}>Add Source</Button>
			</div>

			{showForm && (
				<form
					className="mt-6 space-y-4 rounded-xl border bg-card p-5"
					onSubmit={form.handleSubmit((values) =>
						mutation.mutate(
							{ ...values, id: editing?.id },
							{ onSuccess: handleSaveSuccess },
						),
					)}
					noValidate
				>
					<h2 className="font-semibold">
						{editing ? "Edit Source" : "Add RSS Source"}
					</h2>
					<FormField
						id="source-name"
						label="Source name"
						error={form.formState.errors.name?.message}
						registration={form.register("name")}
					/>
					<FormField
						id="source-rss-url"
						label="RSS URL"
						error={form.formState.errors.rssUrl?.message}
						registration={form.register("rssUrl")}
						type="url"
					/>
					{mutation.isError && (
						<p role="alert" className="text-sm text-destructive">
							Could not save source: {mutation.error.message}
						</p>
					)}
					<div className="flex gap-2">
						<Button type="submit" disabled={mutation.isPending}>
							{mutation.isPending ? "Saving..." : "Save Source"}
						</Button>
						<Button
							type="button"
							variant="outline"
							onClick={() => setShowForm(false)}
						>
							Cancel
						</Button>
					</div>
				</form>
			)}

			{sourcesLoading && (
				<output className="mt-8 border-t" aria-busy="true">
					<span className="sr-only">Loading sources</span>
					{["first", "second", "third"].map((row) => (
						<div key={row} className="flex items-center gap-3 border-b py-5">
							<div className="flex-1 space-y-2">
								<Skeleton className="h-4 w-40 max-w-2/3" />
								<Skeleton className="h-3 w-28" />
							</div>
							<Skeleton className="h-9 w-20" />
							<Skeleton className="h-9 w-24" />
						</div>
					))}
				</output>
			)}
			{sourcesError && (
				<ErrorState
					size="compact"
					error={sourcesErrorValue ?? "Sources unavailable"}
					message="Could not load sources. Please try again"
					action={
						<Button onClick={() => void refetchSources()}>Try again</Button>
					}
				/>
			)}
			{activationMutation.isError && (
				<p role="alert" className="mt-4 text-sm text-destructive">
					Could not update source status: {activationMutation.error.message}
				</p>
			)}
			{!sourcesLoading && !sourcesError && allSources.length === 0 && (
				<NoDataState
					size="compact"
					illustration="feed"
					title="No sources yet"
					description="Add an RSS source to start following articles"
					action={<Button onClick={openCreate}>Add Source</Button>}
				/>
			)}
			{!sourcesLoading && !sourcesError && allSources.length > 0 && (
				<div className="mt-8 border-t">
					{allSources.map((source) => (
						<div
							key={source.id}
							className="flex items-center gap-3 border-b py-5"
						>
							<span className="min-w-0 flex-1">
								<Link
									to="/sources/$sourceId"
									params={{ sourceId: source.id }}
									className="block truncate text-sm font-semibold hover:underline"
								>
									{source.name}
								</Link>
								<span className="text-xs text-muted-foreground">
									{source.type.toUpperCase()} ·{" "}
									{source.active ? "Active" : "Inactive"}
								</span>
							</span>
							{source.type === "rss" && (
								<div className="flex gap-2">
									<Button
										variant="outline"
										size="sm"
										onClick={() => openEdit(source)}
									>
										Edit {source.name}
									</Button>
									<Button
										variant="outline"
										size="sm"
										disabled={activationMutation.isPending}
										onClick={() => {
											activationMutation.reset();
											activationMutation.mutate(
												{ id: source.id, active: !source.active },
												{ onSuccess: handleActivationSuccess },
											);
										}}
									>
										{activationMutation.isPending &&
										activationMutation.variables?.id === source.id
											? source.active
												? "Deactivating..."
												: "Activating..."
											: source.active
												? `Inactive ${source.name}`
												: `Activate ${source.name}`}
									</Button>
								</div>
							)}
						</div>
					))}
				</div>
			)}
		</section>
	);
}
