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

const sourceTypes = [
	{ value: "all", label: "All types" },
	{ value: "rss", label: "RSS" },
	{ value: "youtube", label: "YouTube" },
	{ value: "manual", label: "Manual" },
];

export function SourcesSearch({
	onSearch,
}: {
	onSearch: (filters: SourceFilters) => void;
}) {
	const [name, setName] = useState("");
	const [type, setType] = useState("all");
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
