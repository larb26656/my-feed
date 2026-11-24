import { useState } from "react";
import { toastManager } from "#/components/ui/toaster";
import type { Source } from "#/features/feed/types";
import { useSetSourceActive } from "#/hooks/queries";
import { ProfileImageDialog } from "./ProfileImageDialog";
import type { SourceActionHandlers } from "./SourceActions";
import { SourceDialog } from "./SourceDialog";

export function useSourceActions() {
	const [imageSource, setImageSource] = useState<Source | null>(null);
	const [editing, setEditing] = useState<Source | null>(null);
	const [showForm, setShowForm] = useState(false);
	const mutation = useSetSourceActive();
	const actions: SourceActionHandlers = {
		activationPending: mutation.isPending,
		activationSourceId: mutation.variables?.id,
		onEdit: (source) => {
			setEditing(source);
			setShowForm(true);
		},
		onImage: setImageSource,
		onToggleActive: (source) => {
			mutation.reset();
			mutation.mutate(
				{ id: source.id, active: !source.active },
				{
					onSuccess: (updated) =>
						toastManager.add({
							title: updated.active ? "Source activated" : "Source deactivated",
							type: "success",
						}),
				},
			);
		},
	};
	return {
		actions,
		openCreate: () => {
			setEditing(null);
			setShowForm(true);
		},
		dialogs: (
			<>
				{imageSource && (
					<ProfileImageDialog
						source={imageSource}
						onClose={() => setImageSource(null)}
					/>
				)}
				{showForm && (
					<SourceDialog editing={editing} onClose={() => setShowForm(false)} />
				)}
				{mutation.isError && (
					<p role="alert" className="mt-4 text-sm text-destructive">
						Could not update source status: {mutation.error.message}
					</p>
				)}
			</>
		),
	};
}
