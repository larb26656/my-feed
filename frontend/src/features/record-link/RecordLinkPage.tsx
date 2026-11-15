import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import { FormField } from "#/components/ui/form-field";
import { toastManager } from "#/components/ui/toaster";
import { useSaveLink } from "#/hooks/queries";

const recordLinkSchema = z.object({
	title: z.string().trim().min(1, "Please enter an entry title"),
	link: z.url("Please enter a valid link"),
	description: z.string().trim(),
	thumbnail: z.union([z.literal(""), z.url("Please enter a valid image URL")]),
	publishedAt: z
		.string()
		.refine(
			(value) =>
				!value || z.iso.datetime({ local: true }).safeParse(value).success,
			"Please enter a valid publication date",
		),
});

type RecordLinkForm = z.infer<typeof recordLinkSchema>;

export function RecordLinkPage() {
	const form = useForm<RecordLinkForm>({
		resolver: zodResolver(recordLinkSchema),
		defaultValues: {
			title: "",
			link: "",
			description: "",
			thumbnail: "",
			publishedAt: "",
		},
	});
	const mutation = useSaveLink();
	const submit = (values: RecordLinkForm) => {
		const { thumbnail, publishedAt, ...required } = values;
		mutation.mutate(
			{
				...required,
				...(thumbnail ? { thumbnail } : {}),
				...(publishedAt
					? { publishedAt: new Date(publishedAt).toISOString() }
					: {}),
			},
			{
				onSuccess: () => {
					toastManager.add({ title: "Link saved", type: "success" });
					form.reset();
				},
				onError: (error) => {
					toastManager.add({
						title: "Could not save link",
						description: error.message,
						type: "error",
					});
				},
			},
		);
	};

	return (
		<section>
			<h1 className="sr-only">Save Link</h1>
			<p className="mt-2 text-sm text-muted-foreground">
				Add a link as an entry in the “Saved links” source
			</p>
			<form
				className="mt-6 space-y-4 rounded-xl border bg-card p-5"
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<FormField
					id="entry-title"
					label="Entry title"
					error={form.formState.errors.title?.message}
					registration={form.register("title")}
				/>
				<FormField
					id="entry-link"
					label="Link"
					type="url"
					error={form.formState.errors.link?.message}
					registration={form.register("link")}
				/>
				<FormField
					id="entry-description"
					label="Description"
					error={form.formState.errors.description?.message}
					registration={form.register("description")}
				/>
				<FormField
					id="entry-thumbnail"
					label="Image URL (optional)"
					type="url"
					error={form.formState.errors.thumbnail?.message}
					registration={form.register("thumbnail")}
				/>
				<FormField
					id="entry-published-at"
					label="Publication date (optional)"
					type="datetime-local"
					error={form.formState.errors.publishedAt?.message}
					registration={form.register("publishedAt")}
				/>
				{mutation.isError && (
					<p role="alert" className="text-sm text-destructive">
						Could not save link: {mutation.error.message}
					</p>
				)}
				<Button type="submit" disabled={mutation.isPending}>
					{mutation.isPending ? "Saving..." : "Save Link"}
				</Button>
			</form>
		</section>
	);
}
