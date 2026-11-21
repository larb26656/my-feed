import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "#/components/ui/dialog";
import { FormField } from "#/components/ui/form-field";
import { toastManager } from "#/components/ui/toaster";
import type { Source } from "#/features/feed/types";
import { useSaveSource } from "#/hooks/queries";

const sourceSchema = z
	.object({
		name: z.string().trim().min(1, "Please enter a source name"),
		type: z.enum(["rss", "youtube"]),
		rssUrl: z.string(),
		youtubeChannelId: z.string(),
	})
	.superRefine((value, ctx) => {
		if (value.type === "rss" && !z.url().safeParse(value.rssUrl).success)
			ctx.addIssue({
				code: "custom",
				path: ["rssUrl"],
				message: "Please enter a valid RSS / Atom URL",
			});
		if (
			value.type === "youtube" &&
			!/^(?:https:\/\/www\.youtube\.com\/channel\/)?UC[A-Za-z0-9_-]{22}$/.test(
				value.youtubeChannelId.trim(),
			)
		)
			ctx.addIssue({
				code: "custom",
				path: ["youtubeChannelId"],
				message:
					"Enter a channel ID starting with UC or a canonical YouTube channel URL",
			});
	});
type SourceForm = z.infer<typeof sourceSchema>;

export function SourceDialog({
	editing,
	onClose,
}: {
	editing: Source | null;
	onClose: () => void;
}) {
	const form = useForm<SourceForm>({
		resolver: zodResolver(sourceSchema),
		defaultValues: {
			name: editing?.name ?? "",
			type: editing?.type === "youtube" ? "youtube" : "rss",
			rssUrl: editing?.rssUrl ?? "",
			youtubeChannelId: editing?.youtubeChannelId ?? "",
		},
	});
	const mutation = useSaveSource();
	const handleSaveSuccess = () => {
		toastManager.add({
			title: editing ? "Source updated" : "Source added",
			type: "success",
		});
		onClose();
	};
	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) onClose();
			}}
		>
			<DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
				<DialogHeader>
					<DialogTitle>{editing ? "Edit source" : "Add source"}</DialogTitle>
					<DialogDescription>
						Follow a feed and keep new articles in one place.
					</DialogDescription>
				</DialogHeader>
				<form
					className="flex flex-col gap-5"
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
					{!editing && (
						<fieldset>
							<legend className="mb-2 text-sm font-medium">Source type</legend>
							<div className="flex gap-4">
								<label>
									<input type="radio" value="rss" {...form.register("type")} />{" "}
									RSS feed
								</label>
								<label>
									<input
										type="radio"
										value="youtube"
										{...form.register("type")}
									/>{" "}
									YouTube
								</label>
							</div>
						</fieldset>
					)}
					{form.watch("type") === "rss" ? (
						<FormField
							id="source-rss-url"
							label="RSS / Atom feed URL"
							error={form.formState.errors.rssUrl?.message}
							registration={form.register("rssUrl")}
							type="url"
						/>
					) : editing ? (
						<p className="text-sm">
							YouTube channel ID: {editing.youtubeChannelId}
						</p>
					) : (
						<>
							<FormField
								id="source-youtube-id"
								label="YouTube channel ID or URL"
								error={form.formState.errors.youtubeChannelId?.message}
								registration={form.register("youtubeChannelId")}
							/>
							<p className="text-sm text-muted-foreground">
								Enter a channel ID starting with UC or a canonical
								https://www.youtube.com/channel/ URL. Handles and custom URLs
								are not supported.
							</p>
						</>
					)}
					{mutation.isError && (
						<p role="alert" className="text-sm text-destructive">
							Could not save source: {mutation.error.message}
						</p>
					)}
					<div className="flex flex-col-reverse gap-2 pt-4 sm:flex-row sm:justify-end">
						<Button type="button" variant="outline" onClick={() => onClose()}>
							Cancel
						</Button>
						<Button type="submit" disabled={mutation.isPending}>
							{mutation.isPending ? "Saving..." : "Save source"}
						</Button>
					</div>
				</form>
			</DialogContent>
		</Dialog>
	);
}
