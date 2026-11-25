import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";

export function FeedSort({
	sortDirection,
	setSortDirection,
}: {
	sortDirection: "asc" | "desc";
	setSortDirection: (direction: "asc" | "desc") => void;
}) {
	return (
		<Select
			value={sortDirection}
			onValueChange={(value) => {
				if (value) setSortDirection(value as "asc" | "desc");
			}}
			items={[
				{ value: "desc", label: "Newest first" },
				{ value: "asc", label: "Oldest first" },
			]}
		>
			<SelectTrigger aria-label="Sort entries" variant="ghost">
				<SelectValue />
			</SelectTrigger>
			<SelectContent>
				<SelectGroup>
					<SelectItem value="desc">Newest first</SelectItem>
					<SelectItem value="asc">Oldest first</SelectItem>
				</SelectGroup>
			</SelectContent>
		</Select>
	);
}
