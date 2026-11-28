import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { toastManager } from "#/components/ui/toaster";
import { useSaveLink } from "#/hooks/queries";

const recordLinkSchema = z.object({
	title: z.string().trim().min(1, "Please enter an entry title"),
	link: z
		.string()
		.trim()
		.refine(
			(value) => !value || z.url().safeParse(value).success,
			"Please enter a valid link",
		),
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
				Add an entry to the “Saved links” source
			</p>
			<form
				className="mt-6 rounded-xl border bg-card p-5"
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<FieldGroup>
					<Controller
						name="title"
						control={form.control}
						render={({ field, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel htmlFor="entry-title">Entry title</FieldLabel>
								<Input
									{...field}
									id="entry-title"
									aria-invalid={fieldState.invalid}
									aria-describedby={
										fieldState.invalid ? "entry-title-error" : undefined
									}
								/>
								{fieldState.invalid && (
									<FieldError
										id="entry-title-error"
										errors={[fieldState.error]}
									/>
								)}
							</Field>
						)}
					/>
					<Controller
						name="link"
						control={form.control}
						render={({ field, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel htmlFor="entry-link">Link (optional)</FieldLabel>
								<Input
									{...field}
									id="entry-link"
									type="url"
									aria-invalid={fieldState.invalid}
									aria-describedby={
										fieldState.invalid ? "entry-link-error" : undefined
									}
								/>
								{fieldState.invalid && (
									<FieldError
										id="entry-link-error"
										errors={[fieldState.error]}
									/>
								)}
							</Field>
						)}
					/>
					<Controller
						name="description"
						control={form.control}
						render={({ field, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel htmlFor="entry-description">Description</FieldLabel>
								<Input
									{...field}
									id="entry-description"
									aria-invalid={fieldState.invalid}
									aria-describedby={
										fieldState.invalid ? "entry-description-error" : undefined
									}
								/>
								{fieldState.invalid && (
									<FieldError
										id="entry-description-error"
										errors={[fieldState.error]}
									/>
								)}
							</Field>
						)}
					/>
					<Controller
						name="thumbnail"
						control={form.control}
						render={({ field, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel htmlFor="entry-thumbnail">
									Image URL (optional)
								</FieldLabel>
								<Input
									{...field}
									id="entry-thumbnail"
									type="url"
									aria-invalid={fieldState.invalid}
									aria-describedby={
										fieldState.invalid ? "entry-thumbnail-error" : undefined
									}
								/>
								{fieldState.invalid && (
									<FieldError
										id="entry-thumbnail-error"
										errors={[fieldState.error]}
									/>
								)}
							</Field>
						)}
					/>
					<Controller
						name="publishedAt"
						control={form.control}
						render={({ field, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel htmlFor="entry-published-at">
									Publication date (optional)
								</FieldLabel>
								<Input
									{...field}
									id="entry-published-at"
									type="datetime-local"
									aria-invalid={fieldState.invalid}
									aria-describedby={
										fieldState.invalid ? "entry-published-at-error" : undefined
									}
								/>
								{fieldState.invalid && (
									<FieldError
										id="entry-published-at-error"
										errors={[fieldState.error]}
									/>
								)}
							</Field>
						)}
					/>
					{mutation.isError && (
						<p role="alert" className="text-sm text-destructive">
							Could not save link: {mutation.error.message}
						</p>
					)}
					<Button type="submit" disabled={mutation.isPending}>
						{mutation.isPending ? "Saving..." : "Save Link"}
					</Button>
				</FieldGroup>
			</form>
		</section>
	);
}
