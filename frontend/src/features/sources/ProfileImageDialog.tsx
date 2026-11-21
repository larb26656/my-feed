import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "#/components/ui/dialog";
import type { Source } from "#/features/feed/types";
import { api } from "#/lib/api";
import { sourceKeys } from "#/lib/query-keys";
import { ImageCropPicker, prepareImage, useImageCrop } from "./ImageCropPicker";

export function ProfileImageDialog({
	source,
	onClose,
}: {
	source: Source;
	onClose: () => void;
}) {
	const queryClient = useQueryClient();
	const cropState = useImageCrop();
	const { image, area, error, setError } = cropState;
	const [pending, setPending] = useState(false);
	async function save() {
		if (!image || !area) return;
		setPending(true);
		setError("");
		try {
			const blob = await prepareImage(image, area);
			await api(`/sources/${source.id}/profile-image`, {
				method: "PUT",
				headers: { "Content-Type": "image/jpeg" },
				body: blob,
			});
			await queryClient.invalidateQueries({ queryKey: sourceKeys.all });
			onClose();
		} catch (e) {
			setError(
				e instanceof Error
					? e.message
					: "Could not save image. Please try again.",
			);
		} finally {
			setPending(false);
		}
	}
	async function remove() {
		setPending(true);
		setError("");
		try {
			await api(`/sources/${source.id}/profile-image`, { method: "DELETE" });
			await queryClient.invalidateQueries({ queryKey: sourceKeys.all });
			onClose();
		} catch (e) {
			setError(
				e instanceof Error
					? e.message
					: "Could not remove image. Please try again.",
			);
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
					<DialogTitle>Change profile image</DialogTitle>
					<DialogDescription>
						Choose and frame an image for {source.name}.
					</DialogDescription>
				</DialogHeader>
				<ImageCropPicker
					state={cropState}
					disabled={pending}
					name={source.name}
					sourceId={source.id}
					imageId={source.profileImageId}
				/>
				{error && (
					<p role="alert" className="text-destructive">
						{error}
					</p>
				)}
				<div className="flex justify-end gap-2">
					{source.profileImageId && (
						<Button
							variant="outline"
							onClick={() => void remove()}
							disabled={pending}
						>
							Remove image
						</Button>
					)}
					<Button variant="outline" onClick={onClose} disabled={pending}>
						Cancel
					</Button>
					<Button
						onClick={() => void save()}
						disabled={!image || !area || pending}
					>
						{pending ? "Saving..." : "Save image"}
					</Button>
				</div>
			</DialogContent>
		</Dialog>
	);
}
