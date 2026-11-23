import { type Control, Controller } from "react-hook-form";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldLabel,
} from "#/components/ui/field";
import { Input } from "#/components/ui/input";

export type SourceTypeDefinition = {
	code: string;
	label: string;
	schema: Record<string, unknown>;
	ui: Record<
		string,
		{ label: string; widget: string; description?: string; error?: string }
	>;
};

export function sourceConfigDefaults(type: SourceTypeDefinition) {
	const properties = type.schema.properties as
		| Record<string, { default?: unknown }>
		| undefined;
	return Object.fromEntries(
		Object.entries(properties ?? {})
			.filter(([, property]) => property.default !== undefined)
			.map(([key, property]) => [key, property.default]),
	);
}

export function supportsSourceConfigFields(
	schema: Record<string, unknown>,
	ui: SourceTypeDefinition["ui"],
) {
	const properties = schema.properties;
	if (
		Object.keys(schema).some(
			(key) =>
				![
					"$schema",
					"type",
					"additionalProperties",
					"properties",
					"required",
					"title",
					"description",
					"allOf",
				].includes(key),
		)
	)
		return false;
	if (
		schema.type !== "object" ||
		!properties ||
		typeof properties !== "object" ||
		Array.isArray(properties)
	)
		return false;
	if (["anyOf", "oneOf", "if", "then", "else"].some((key) => key in schema))
		return false;
	return Object.entries(properties).every(([key, raw]) => {
		if (!raw || typeof raw !== "object" || Array.isArray(raw)) return false;
		const field = raw as Record<string, unknown>;
		if (
			Object.keys(field).some(
				(key) =>
					![
						"type",
						"enum",
						"pattern",
						"minimum",
						"maximum",
						"minLength",
						"maxLength",
						"default",
						"title",
						"description",
					].includes(key),
			)
		)
			return false;
		if (
			Object.keys(ui[key] ?? {}).some(
				(key) => !["label", "widget", "description", "error"].includes(key),
			) ||
			!ui[key]?.label
		)
			return false;
		const widget = ui[key]?.widget;
		if (widget === "text" || widget === "url") return field.type === "string";
		if (widget === "number")
			return field.type === "number" || field.type === "integer";
		if (widget === "checkbox") return field.type === "boolean";
		if (widget === "select")
			return (
				["string", "number", "integer"].includes(String(field.type)) &&
				Array.isArray(field.enum)
			);
		return false;
	});
}

type FormValues = {
	name: string;
	type: string;
	config: Record<string, unknown>;
};
export function SourceConfigFields({
	definition,
	control,
	clearPreview,
	readOnly = false,
}: {
	definition: SourceTypeDefinition;
	control: Control<FormValues>;
	clearPreview: () => void;
	readOnly?: boolean;
}) {
	const properties = definition.schema.properties as
		| Record<
				string,
				{ type?: string; enum?: (string | number)[]; description?: string }
		  >
		| undefined;
	if (
		!properties ||
		!supportsSourceConfigFields(definition.schema, definition.ui)
	)
		return <p role="alert">This Source Type configuration is unsupported</p>;
	return (
		<>
			{Object.entries(properties).map(([key, property]) => {
				const meta = definition.ui[key];
				const id = `source-config-${key}`;
				const required =
					Array.isArray(definition.schema.required) &&
					definition.schema.required.includes(key);
				const errorId = `${id}-error`;
				return (
					<Controller
						key={key}
						name={`config.${key}` as `config.${string}`}
						control={control}
						render={({ field, fieldState }) => (
							<Field data-invalid={fieldState.invalid}>
								<FieldLabel
									htmlFor={id}
									className={required ? "after:content-['*']" : undefined}
								>
									{meta.label}
								</FieldLabel>
								{meta.widget === "select" ? (
									<select
										id={id}
										aria-required={required}
										aria-invalid={fieldState.invalid}
										aria-describedby={fieldState.invalid ? errorId : undefined}
										aria-readonly={readOnly || undefined}
										disabled={readOnly}
										value={String(field.value ?? "")}
										onChange={(event) => {
											field.onChange(
												(property.type === "number" ||
													property.type === "integer") &&
													event.target.value !== ""
													? Number(event.target.value)
													: event.target.value,
											);
											clearPreview();
										}}
									>
										<option value="">Select...</option>
										{property.enum?.map((value) => (
											<option key={String(value)} value={String(value)}>
												{String(value)}
											</option>
										))}
									</select>
								) : meta.widget === "checkbox" ? (
									<input
										id={id}
										aria-required={required}
										aria-invalid={fieldState.invalid}
										aria-describedby={fieldState.invalid ? errorId : undefined}
										type="checkbox"
										checked={Boolean(field.value)}
										disabled={readOnly}
										readOnly={readOnly}
										onChange={(event) => {
											field.onChange(event.target.checked);
											clearPreview();
										}}
									/>
								) : (
									<Input
										id={id}
										aria-required={required}
										aria-invalid={fieldState.invalid}
										aria-describedby={fieldState.invalid ? errorId : undefined}
										type={
											meta.widget === "number"
												? "number"
												: meta.widget === "url"
													? "url"
													: "text"
										}
										value={String(field.value ?? "")}
										readOnly={readOnly}
										onChange={(event) => {
											field.onChange(
												meta.widget === "number" && event.target.value !== ""
													? Number(event.target.value)
													: event.target.value,
											);
											clearPreview();
										}}
									/>
								)}
								{(meta.description || property.description) && (
									<FieldDescription>
										{meta.description || property.description}
									</FieldDescription>
								)}
								{fieldState.invalid && (
									<FieldError id={errorId} errors={[fieldState.error]} />
								)}
							</Field>
						)}
					/>
				);
			})}
		</>
	);
}
