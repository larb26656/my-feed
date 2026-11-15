import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "#/components/ui/button";
import { FormField } from "#/components/ui/form-field";
import { toastManager } from "#/components/ui/toaster";
import { useSaveLink } from "#/hooks/queries";

const recordLinkSchema = z.object({
	title: z.string().trim().min(1, "กรุณาระบุชื่อ Entry"),
	link: z.url("กรุณาระบุลิงก์ที่ถูกต้อง"),
	description: z.string().trim(),
	thumbnail: z.union([z.literal(""), z.url("กรุณาระบุ URL รูปภาพที่ถูกต้อง")]),
	publishedAt: z
		.string()
		.refine(
			(value) =>
				!value || z.iso.datetime({ local: true }).safeParse(value).success,
			"กรุณาระบุวันที่เผยแพร่ที่ถูกต้อง",
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
					toastManager.add({ title: "บันทึกลิงก์แล้ว", type: "success" });
					form.reset();
				},
				onError: (error) => {
					toastManager.add({
						title: "บันทึกลิงก์ไม่สำเร็จ",
						description: error.message,
						type: "error",
					});
				},
			},
		);
	};

	return (
		<section>
			<h1 className="text-2xl font-bold">บันทึกลิงก์</h1>
			<p className="mt-2 text-sm text-muted-foreground">
				เพิ่มลิงก์เป็น Entry ใน Source “Saved links”
			</p>
			<form
				className="mt-6 space-y-4 rounded-xl border bg-card p-5"
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<FormField
					id="entry-title"
					label="ชื่อ Entry"
					error={form.formState.errors.title?.message}
					registration={form.register("title")}
				/>
				<FormField
					id="entry-link"
					label="ลิงก์"
					type="url"
					error={form.formState.errors.link?.message}
					registration={form.register("link")}
				/>
				<FormField
					id="entry-description"
					label="คำอธิบาย"
					error={form.formState.errors.description?.message}
					registration={form.register("description")}
				/>
				<FormField
					id="entry-thumbnail"
					label="URL รูปภาพ (ไม่บังคับ)"
					type="url"
					error={form.formState.errors.thumbnail?.message}
					registration={form.register("thumbnail")}
				/>
				<FormField
					id="entry-published-at"
					label="วันที่เผยแพร่ (ไม่บังคับ)"
					type="datetime-local"
					error={form.formState.errors.publishedAt?.message}
					registration={form.register("publishedAt")}
				/>
				{mutation.isError && (
					<p role="alert" className="text-sm text-destructive">
						บันทึกลิงก์ไม่สำเร็จ: {mutation.error.message}
					</p>
				)}
				<Button type="submit" disabled={mutation.isPending}>
					{mutation.isPending ? "กำลังบันทึก..." : "บันทึกลิงก์"}
				</Button>
			</form>
		</section>
	);
}
