import { zodResolver } from "@hookform/resolvers/zod";
import { Link } from "@tanstack/react-router";
import { Rss, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ErrorState } from "#/components/ErrorState";
import { NoDataState } from "#/components/NoDataState";
import { SourceAvatar } from "#/components/SourceAvatar";
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
	useEffect(() => {
		if (!showForm) return;
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key === "Escape") setShowForm(false);
		};
		window.addEventListener("keydown", onKeyDown);
		return () => window.removeEventListener("keydown", onKeyDown);
	}, [showForm]);
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
					<h1 className="sr-only">Your Sources</h1>
					<p className="mt-2 text-sm text-muted-foreground">
						Choose a source to view its articles
					</p>
				</div>
				<Button onClick={openCreate}>Add Source</Button>
			</div>

			{showForm && (
				<div
					className="fixed inset-0 z-40 flex items-end justify-center bg-black/45 sm:items-center sm:p-6"
					onMouseDown={(event) => {
						if (event.target === event.currentTarget) setShowForm(false);
					}}
				>
					<section
						role="dialog"
						aria-modal="true"
						aria-labelledby="source-dialog-title"
						className="max-h-[90dvh] w-full overflow-y-auto rounded-t-3xl border bg-background shadow-2xl sm:max-w-lg sm:rounded-2xl"
					>
						<div className="mx-auto mt-3 h-1 w-10 rounded-full bg-muted-foreground/30 sm:hidden" />
						<div className="flex items-start justify-between border-b px-6 py-5 sm:px-7">
							<div>
								<h2 id="source-dialog-title" className="text-xl font-semibold">
									{editing ? "Edit RSS source" : "Add RSS source"}
								</h2>
								<p className="mt-1 text-sm text-muted-foreground">
									Follow a feed and keep new articles in one place.
								</p>
							</div>
							<button
								type="button"
								aria-label="Close"
								onClick={() => setShowForm(false)}
								className="rounded-lg p-2 text-muted-foreground hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<X size={18} />
							</button>
						</div>
						<form
							className="space-y-5 px-6 pb-6 pt-5 sm:px-7 sm:pb-7"
							onSubmit={form.handleSubmit((values) =>
								mutation.mutate(
									{ ...values, id: editing?.id },
									{ onSuccess: handleSaveSuccess },
								),
							)}
							noValidate
						>
							<FormField
								id="source-name"
								label="Source name"
								error={form.formState.errors.name?.message}
								registration={form.register("name")}
							/>
							<fieldset>
								<legend className="mb-2 text-sm font-medium">Source type</legend>
								<button
									type="button"
									aria-pressed="true"
									disabled
									className="flex w-full cursor-default items-center gap-3 rounded-xl border border-primary bg-primary/5 p-4 text-left ring-1 ring-primary disabled:opacity-100"
								>
									<span className="grid size-10 place-items-center rounded-lg bg-muted text-primary">
										<Rss size={18} />
									</span>
									<span className="flex-1">
										<strong className="block text-sm">RSS feed</strong>
										<span className="text-xs text-muted-foreground">
											A website or publication feed
										</span>
									</span>
									<span className="rounded-full bg-muted px-2 py-1 text-[10px] text-muted-foreground">
										Available
									</span>
								</button>
							</fieldset>
							<FormField
								id="source-rss-url"
								label="RSS feed URL"
								error={form.formState.errors.rssUrl?.message}
								registration={form.register("rssUrl")}
								type="url"
							/>
							{mutation.isError && (
								<p role="alert" className="text-sm text-destructive">
									Could not save source: {mutation.error.message}
								</p>
							)}
							<div className="flex flex-col-reverse gap-2 border-t pt-4 sm:flex-row sm:justify-end">
								<Button type="button" variant="outline" onClick={() => setShowForm(false)}>
									Cancel
								</Button>
								<Button type="submit" disabled={mutation.isPending}>
									{mutation.isPending ? "Saving..." : "Save source"}
								</Button>
							</div>
						</form>
					</section>
				</div>
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
							<SourceAvatar name={source.name} image={source.image} size={40} />
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
