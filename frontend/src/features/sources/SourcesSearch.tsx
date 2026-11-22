import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import type { SourceFilters } from "#/hooks/queries";
import { api } from "#/lib/api";

export function SourcesSearch({
	initialName = "",
	initialType = "all",
	onSearch,
}: {
	initialName?: string;
	initialType?: string;
	onSearch: (filters: SourceFilters) => void;
}) {
	const { data } = useQuery({
		queryKey: ["source-types"],
		queryFn: async () =>
			(await api<{ types: { code: string; label: string }[] }>("/source-types"))
				.data.types,
	});
	const sourceTypes = [
		{ value: "all", label: "All types" },
		...(data ?? []).map((item) => ({ value: item.code, label: item.label })),
		{ value: "manual", label: "Manual" },
	];
	const [name, setName] = useState(initialName);
	const [type, setType] = useState(initialType);
	return (
		<form
			className="mt-4 flex flex-wrap items-end gap-3"
			onSubmit={(event) => {
				event.preventDefault();
				onSearch({ name, type: type === "all" ? "" : type });
			}}
		>
			<div className="flex min-w-48 flex-1 flex-col gap-1.5">
				<Label htmlFor="source-search">Search by name</Label>
				<Input
					id="source-search"
					type="search"
					placeholder="Search sources"
					value={name}
					onChange={(event) => setName(event.target.value)}
				/>
			</div>
			<div className="flex flex-col gap-1.5">
				<Label htmlFor="source-type">Type</Label>
				<Select
					items={sourceTypes}
					value={type}
					onValueChange={(value) => {
						if (value !== null) setType(value);
					}}
				>
					<SelectTrigger id="source-type">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectGroup>
							{sourceTypes.map((item) => (
								<SelectItem key={item.value} value={item.value}>
									{item.label}
								</SelectItem>
							))}
						</SelectGroup>
					</SelectContent>
				</Select>
			</div>
			<Button type="submit">Search</Button>
		</form>
	);
}
