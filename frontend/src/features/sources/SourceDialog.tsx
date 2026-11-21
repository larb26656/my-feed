import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "#/components/ui/dialog";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { toastManager } from "#/components/ui/toaster";
import type { Source } from "#/features/feed/types";
import { useSaveSource } from "#/hooks/queries";
import { api } from "#/lib/api";
import { sourceKeys } from "#/lib/query-keys";
import { ImageCropPicker, prepareImage, useImageCrop } from "./ImageCropPicker";

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
	const queryClient = useQueryClient();
	const cropState = useImageCrop();
	const [created, setCreated] = useState<Source | null>(null);
	const [uploadError, setUploadError] = useState("");
	const [pending, setPending] = useState(false);

	const handleSaveSuccess = () => {
		toastManager.add({
			title: editing ? "Source updated" : "Source added",
			type: "success",
		});
		onClose();
	};
	async function save(values: SourceForm) {
		setPending(true);
		setUploadError("");
		let sourceCreated = Boolean(created);
		try {
			const source =
				created ?? (await mutation.mutateAsync({ ...values, id: editing?.id }));
			if (!editing && (cropState.image || created)) {
				setCreated(source);
				sourceCreated = true;
				if (!cropState.image)
					throw new Error("Please choose an image to retry.");
				if (!cropState.area)
					throw new Error("Please wait for the crop preview.");
				const blob = await prepareImage(cropState.image, cropState.area);
				await api(`/sources/${source.id}/profile-image`, {
					method: "PUT",
					headers: { "Content-Type": "image/jpeg" },
					body: blob,
				});
				await queryClient.invalidateQueries({ queryKey: sourceKeys.all });
			}
			handleSaveSuccess();
		} catch (error) {
			if (sourceCreated) {
				setUploadError(
					error instanceof Error ? error.message : "Please try again.",
				);
			}
		} finally {
			setPending(false);
		}
	}
	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open && !pending) onClose();
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
					onSubmit={form.handleSubmit((values) => void save(values))}
					noValidate
				>
					{!editing && (
						<ImageCropPicker
							state={cropState}
							disabled={pending}
							name={form.watch("name")}
						/>
					)}
					<fieldset disabled={Boolean(created) || pending} className="contents">
						<FieldGroup>
							<Controller
								name="name"
								control={form.control}
								render={({ field, fieldState }) => (
									<Field data-invalid={fieldState.invalid}>
										<FieldLabel htmlFor="source-name">Source name</FieldLabel>
										<Input
											{...field}
											id="source-name"
											aria-invalid={fieldState.invalid}
											aria-describedby={
												fieldState.invalid ? "source-name-error" : undefined
											}
										/>

										{fieldState.invalid && (
											<FieldError
												id="source-name-error"
												errors={[fieldState.error]}
											/>
										)}
									</Field>
								)}
							/>
							{!editing && (
								<fieldset>
									<legend className="mb-2 text-sm font-medium">
										Source type
									</legend>
									<div className="flex gap-4">
										<label>
											<input
												type="radio"
												value="rss"
												{...form.register("type")}
											/>{" "}
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
								<Controller
									name="rssUrl"
									control={form.control}
									render={({ field, fieldState }) => (
										<Field data-invalid={fieldState.invalid}>
											<FieldLabel htmlFor="source-rss-url">
												RSS / Atom feed URL
											</FieldLabel>
											<Input
												{...field}
												id="source-rss-url"
												type="url"
												aria-invalid={fieldState.invalid}
												aria-describedby={
													fieldState.invalid
														? "source-rss-url-error"
														: undefined
												}
											/>

											{fieldState.invalid && (
												<FieldError
													id="source-rss-url-error"
													errors={[fieldState.error]}
												/>
											)}
										</Field>
									)}
								/>
							) : editing ? (
								<p className="text-sm">
									YouTube channel ID: {editing.youtubeChannelId}
								</p>
							) : (
								<>
									<Controller
										name="youtubeChannelId"
										control={form.control}
										render={({ field, fieldState }) => (
											<Field data-invalid={fieldState.invalid}>
												<FieldLabel htmlFor="source-youtube-id">
													YouTube channel ID or URL
												</FieldLabel>
												<Input
													{...field}
													id="source-youtube-id"
													aria-invalid={fieldState.invalid}
													aria-describedby={
														fieldState.invalid
															? "source-youtube-id-error"
															: undefined
													}
												/>
												<FieldDescription>
													Enter a channel ID starting with UC or a canonical
													https://www.youtube.com/channel/ URL. Handles and
													custom URLs are not supported.
												</FieldDescription>
												{fieldState.invalid && (
													<FieldError
														id="source-youtube-id-error"
														errors={[fieldState.error]}
													/>
												)}
											</Field>
										)}
									/>
								</>
							)}
						</FieldGroup>
					</fieldset>

					{cropState.error && (
						<p role="alert" className="text-destructive">
							{cropState.error}
						</p>
					)}
					{uploadError && (
						<p role="alert" className="text-destructive">
							Source was created, but the image could not be uploaded:{" "}
							{uploadError} You can retry the upload.
						</p>
					)}
					{mutation.isError && !created && (
						<p role="alert" className="text-sm text-destructive">
							Could not save source: {mutation.error.message}
						</p>
					)}
					<div className="flex flex-col-reverse gap-2 pt-4 sm:flex-row sm:justify-end">
						<Button
							type="button"
							variant="outline"
							disabled={pending}
							onClick={() => onClose()}
						>
							Cancel
						</Button>
						<Button
							type="submit"
							disabled={pending || Boolean(cropState.image && !cropState.area)}
						>
							{pending
								? "Saving..."
								: created
									? "Retry image upload"
									: "Save source"}
						</Button>
					</div>
				</form>
			</DialogContent>
		</Dialog>
	);
}
