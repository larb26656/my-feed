import { ajvResolver } from "@hookform/resolvers/ajv";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import Ajv2020 from "ajv/dist/2020";
import { useEffect, useRef, useState } from "react";
import {
	Controller,
	type FieldErrors,
	type Resolver,
	useForm,
} from "react-hook-form";
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
	FieldError,
	FieldGroup,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";

import { toastManager } from "#/components/ui/toaster";
import { FeedItem } from "#/features/feed/FeedItem";
import type { Source } from "#/features/feed/types";
import { usePreviewSource, useSaveSource } from "#/hooks/queries";
import { api } from "#/lib/api";
import type { components } from "#/lib/api/schema";
import { sourceKeys } from "#/lib/query-keys";
import { ImageCropPicker, prepareImage, useImageCrop } from "./ImageCropPicker";
import {
	SourceConfigFields,
	sourceConfigDefaults,
	supportsSourceConfigFields,
} from "./SourceConfigFields";

type PreviewEntry =
	components["schemas"]["internal_modules_source_adapter_http.previewEntryResponse"];

const sourceSchema = z.object({
	name: z.string().trim().min(1, "Please enter a source name"),
	type: z.string().min(1),
	config: z.record(z.string(), z.any()),
});
type SourceForm = z.infer<typeof sourceSchema>;

export function SourceDialog({
	editing,
	onClose,
}: {
	editing: Source | null;
	onClose: () => void;
}) {
	const sourceTypes = useQuery({
		queryKey: ["source-types"],
		enabled: true,
		queryFn: async () => {
			const { data } = await api<{
				types: {
					code: string;
					label: string;
					schema: Record<string, unknown>;
					ui: Record<
						string,
						{
							label: string;
							widget: string;
							description?: string;
							error?: string;
						}
					>;
				}[];
			}>("/source-types");
			if (!Array.isArray(data.types))
				throw new Error("Invalid Source Type metadata");
			return data.types;
		},
	});
	const previewValidation = useRef(false);
	const resolver: Resolver<SourceForm> = async (values, context, options) => {
		const result = await zodResolver(sourceSchema)(values, context, options);
		const errors: FieldErrors<SourceForm> = result.errors;
		const changed =
			!editing ||
			JSON.stringify(values.config) !== JSON.stringify(editing.config ?? {});
		if (changed || previewValidation.current) {
			const definition = sourceTypes.data?.find(
				(item) => item.code === values.type,
			);
			try {
				if (
					!definition ||
					!supportsSourceConfigFields(definition.schema, definition.ui)
				)
					throw new Error("unsupported");
				// The resolver package uses AJV's Draft 7 class. Our supported
				// flat keywords have identical semantics in Draft 7 and 2020-12;
				// validate the published schema with AJV 2020-12 as well.
				const validate = new Ajv2020({ allErrors: true }).compile(
					definition.schema,
				);
				const valid2020 = validate(values.config);
				const { $schema: _draft, ...compatibleSchema } = definition.schema;
				const configResolver = ajvResolver(
					{
						type: "object",
						properties: { config: compatibleSchema },
						required: ["config"],
					} as never,
					{ allErrors: true },
					{ mode: "sync" },
				);
				const resolved = await configResolver(values, context, options);
				if (!valid2020 || resolved.errors.config) {
					const fieldErrors: Record<string, { type: string; message: string }> =
						{};
					const resolverErrors = resolved.errors.config as
						| Record<string, { type?: string; message?: string }>
						| undefined;
					for (const [key, error] of Object.entries(resolverErrors ?? {})) {
						const type = error.type ?? "validate";
						fieldErrors[key || "root"] = {
							type,
							message:
								definition.ui[key]?.error ??
								(type === "required"
									? "Required"
									: "Please enter a valid value"),
						};
					}
					// Preserve Draft 2020-12 errors if the resolver's Draft 7 engine
					// ever disagrees with the supported-schema validation.
					if (!valid2020 && Object.keys(fieldErrors).length === 0) {
						for (const error of validate.errors ?? []) {
							const key =
								error.keyword === "required"
									? String(error.params.missingProperty)
									: error.instancePath.slice(1);
							fieldErrors[key || "root"] = {
								type: error.keyword,
								message:
									definition.ui[key]?.error ??
									(error.keyword === "required"
										? "Required"
										: "Please enter a valid value"),
							};
						}
					}
					errors.config = fieldErrors;
				}
			} catch {
				errors.config = {
					root: {
						type: "validate",
						message: "This Source Type configuration is unsupported",
					},
				};
			}
		}
		return Object.keys(errors).length
			? { values: {}, errors }
			: { values: result.values as SourceForm, errors: {} };
	};
	const form = useForm<SourceForm>({
		resolver,
		defaultValues: {
			name: editing?.name ?? "",
			type: editing?.type ?? "",
			config: editing?.config ?? {},
		},
	});
	const selectedType = form.watch("type");
	const definition = sourceTypes.data?.find(
		(item) => item.code === selectedType,
	);
	useEffect(() => {
		if (!editing && sourceTypes.data?.length && !form.getValues("type")) {
			form.setValue("type", sourceTypes.data[0].code);
			form.setValue("config", sourceConfigDefaults(sourceTypes.data[0]));
		}
	}, [sourceTypes.data, editing, form]);
	async function validateConfig() {
		previewValidation.current = true;
		try {
			return await form.trigger("config");
		} finally {
			previewValidation.current = false;
		}
	}

	const mutation = useSaveSource();
	const previewMutation = usePreviewSource();

	const queryClient = useQueryClient();
	const cropState = useImageCrop();
	const [croppedImage, setCroppedImage] = useState<string | null>(null);
	useEffect(() => {
		let active = true;
		let url: string | undefined;
		setCroppedImage(null);
		if (cropState.image && cropState.area) {
			void prepareImage(cropState.image, cropState.area)
				.then((blob) => {
					if (active) {
						url = URL.createObjectURL(blob);
						setCroppedImage(url);
					}
				})
				.catch(() => {});
		}
		return () => {
			active = false;
			if (url) URL.revokeObjectURL(url);
		};
	}, [cropState.image, cropState.area]);
	const [created, setCreated] = useState<Source | null>(null);
	const [uploadError, setUploadError] = useState("");
	const [pending, setPending] = useState(false);
	const [preview, setPreview] = useState<PreviewEntry | null>(null);
	const [previewStatus, setPreviewStatus] = useState<
		"idle" | "pending" | "empty" | "error" | "ready"
	>("idle");
	const [previewError, setPreviewError] = useState("");
	const generation = useRef(0);
	const controller = useRef<AbortController | null>(null);
	useEffect(
		() => () => {
			generation.current++;
			controller.current?.abort();
		},
		[],
	);
	function clearPreview() {
		generation.current++;
		controller.current?.abort();
		setPreview(null);
		setPreviewStatus("idle");
	}
	async function testFetch() {
		if (!(await validateConfig())) return;
		const values = form.getValues();
		clearPreview();
		const current = generation.current;
		const abort = new AbortController();
		controller.current = abort;
		setPreviewStatus("pending");
		try {
			const result = await previewMutation.mutateAsync({
				type: values.type,
				config: values.config,
				signal: abort.signal,
			});
			if (generation.current !== current) return;
			setPreview(result.data.entry);
			setPreviewStatus(result.data.entry ? "ready" : "empty");
		} catch (error) {
			if (generation.current !== current) return;
			setPreviewError(
				error instanceof Error ? error.message : "Please try again.",
			);
			setPreviewStatus("error");
		}
	}

	const handleSaveSuccess = () => {
		toastManager.add({
			title: editing ? "Source updated" : "Source added",
			type: "success",
		});
		onClose();
	};
	async function save(values: SourceForm) {
		const configChanged =
			!editing ||
			JSON.stringify(values.config) !== JSON.stringify(editing.config ?? {});

		setPending(true);
		setUploadError("");
		let sourceCreated = Boolean(created);
		try {
			const source =
				created ??
				(await mutation.mutateAsync({
					...values,
					id: editing?.id,
					includeConfig: configChanged,
				}));
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
					{sourceTypes.isPending && <p>Loading Source Types…</p>}
					{sourceTypes.isError && (
						<p role="alert">Could not load Source Types. Please try again.</p>
					)}
					<fieldset
						disabled={
							Boolean(created) ||
							pending ||
							sourceTypes.isPending ||
							sourceTypes.isError
						}
						className="contents"
					>
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
										{sourceTypes.data?.map((type) => (
											<label key={type.code}>
												<input
													type="radio"
													value={type.code}
													{...form.register("type", {
														onChange: () => {
															form.setValue(
																"config",
																sourceConfigDefaults(type),
															);
															form.clearErrors();
															clearPreview();
														},
													})}
												/>{" "}
												{type.label}
											</label>
										))}
									</div>
								</fieldset>
							)}
							{definition && (
								<SourceConfigFields
									definition={definition}
									control={form.control}
									clearPreview={clearPreview}
								/>
							)}
							{typeof form.formState.errors.config?.root?.message ===
								"string" && (
								<p role="alert">{form.formState.errors.config.root.message}</p>
							)}
						</FieldGroup>
					</fieldset>

					{!editing && !created && (
						<div className="space-y-3">
							<Button
								type="button"
								variant="outline"
								disabled={
									previewStatus === "pending" ||
									sourceTypes.isPending ||
									sourceTypes.isError
								}
								onClick={() => void testFetch()}
							>
								{previewStatus === "pending" ? "Testing..." : "Test fetch"}
							</Button>
							{previewStatus === "empty" && <p>No entries found.</p>}
							{previewStatus === "error" && (
								<p role="alert" className="text-destructive">
									{previewError}
								</p>
							)}
							{preview && (
								<FeedItem
									preview
									entry={preview}
									source={{
										name: form.watch("name") || "Source",
										type: form.watch("type"),
										image: croppedImage,
									}}
								/>
							)}
						</div>
					)}

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
							disabled={
								pending ||
								sourceTypes.isPending ||
								sourceTypes.isError ||
								Boolean(cropState.image && !cropState.area)
							}
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
