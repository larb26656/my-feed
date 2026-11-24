import { useQuery } from "@tanstack/react-query";
import { ListFilter } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
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
import { api } from "#/lib/api";
import { SourceSortMenu } from "./mobile/SourceSortMenu";
import type { SourceFilters, SourceSort } from "./source.api";

const searchDebounceMs = 300;

function useSourceTypeOptions() {
	const { data } = useQuery({
		queryKey: ["source-types"],
		queryFn: async () =>
			(await api<{ types: { code: string; label: string }[] }>("/source-types"))
				.data.types,
	});
	const options = [
		{ value: "all", label: "All types" },
		...(data ?? []).map((item) => ({ value: item.code, label: item.label })),
	];
	if (!options.some((item) => item.value === "manual")) {
		options.push({ value: "manual", label: "Manual" });
	}
	return options;
}

export function SourceToolbar({
	variant,
	initialName,
	initialType,
	sort,
	onSearch,
	onSortChange,
}: {
	variant: "desktop" | "mobile";
	initialName: string;
	initialType: string;
	sort: SourceSort;
	onSearch: (filters: SourceFilters) => void;
	onSortChange: (sort: SourceSort) => void;
}) {
	const sourceTypes = useSourceTypeOptions();
	const [name, setName] = useState(initialName);
	const [type, setType] = useState(initialType);
	const [filterOpen, setFilterOpen] = useState(false);
	const onSearchRef = useRef(onSearch);
	onSearchRef.current = onSearch;
	const typeRef = useRef(type);
	typeRef.current = type;
	const published = useRef(
		`${initialName}\n${initialType === "all" ? "" : initialType}`,
	);

	const publish = useCallback((nextName: string, nextType: string) => {
		const filters = {
			name: nextName,
			type: nextType === "all" ? "" : nextType,
		};
		const key = `${filters.name}\n${filters.type}`;
		if (published.current === key) return;
		published.current = key;
		onSearchRef.current(filters);
	}, []);

	useEffect(() => {
		if (variant !== "mobile") return;
		const handle = window.setTimeout(() => {
			publish(name, typeRef.current);
		}, searchDebounceMs);
		return () => window.clearTimeout(handle);
	}, [name, publish, variant]);

	if (variant === "mobile") {
		return (
			<div className="mt-4 flex items-center gap-2">
				<Input
					type="search"
					aria-label="Search sources"
					placeholder="Search sources..."
					value={name}
					className="min-w-0 flex-1"
					onChange={(event) => setName(event.target.value)}
				/>
				<div className="relative">
					<Button
						type="button"
						variant={type !== "all" ? "secondary" : "outline"}
						size="icon"
						className="size-10"
						aria-label="Filter by type"
						aria-haspopup="menu"
						aria-expanded={filterOpen}
						aria-pressed={type !== "all"}
						onClick={() => setFilterOpen((open) => !open)}
					>
						<ListFilter />
					</Button>
					{filterOpen && (
						<div
							role="menu"
							aria-label="Filter by type"
							className="absolute right-0 top-full z-30 mt-2 min-w-44 rounded-xl border bg-popover p-1 shadow-lg"
						>
							{sourceTypes.map((item) => (
								<button
									key={item.value}
									type="button"
									role="menuitemradio"
									aria-checked={type === item.value}
									className="flex w-full items-center rounded-lg px-3 py-2 text-left text-sm hover:bg-muted aria-checked:bg-muted aria-checked:font-medium"
									onClick={() => {
										setType(item.value);
										setFilterOpen(false);
										publish(name, item.value);
									}}
								>
									{item.label}
								</button>
							))}
						</div>
					)}
				</div>
				<SourceSortMenu sort={sort} onSortChange={onSortChange} />
			</div>
		);
	}

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
