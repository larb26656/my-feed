import { Field } from "@base-ui/react/field";
import type { UseFormRegisterReturn } from "react-hook-form";

type FormFieldProps = {
	id: string;
	label: string;
	error?: string;
	registration: UseFormRegisterReturn;
	type?: string;
};

export function FormField({
	id,
	label,
	error,
	registration,
	type = "text",
}: FormFieldProps) {
	return (
		<Field.Root invalid={Boolean(error)} className="space-y-1.5">
			<Field.Label htmlFor={id} className="text-sm font-medium">
				{label}
			</Field.Label>
			<Field.Control
				id={id}
				type={type}
				aria-invalid={Boolean(error)}
				aria-describedby={error ? `${id}-error` : undefined}
				className="h-control w-full rounded-md border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring aria-invalid:border-destructive"
				{...registration}
			/>
			{error && (
				<Field.Error
					match={true}
					id={`${id}-error`}
					className="text-sm text-destructive"
				>
					{error}
				</Field.Error>
			)}
		</Field.Root>
	);
}
