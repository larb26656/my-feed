import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { cn } from "#/lib/utils";

export function FeedFilters({
	sortField,
	setSortField,
	sortDirection,
	setSortDirection,
	className,
}: {
	sortField: "publishedAtSort" | "createdAtSort" | "savedAtSort";
	setSortField: (
		field: "publishedAtSort" | "createdAtSort" | "savedAtSort",
	) => void;
	sortDirection: "asc" | "desc";
	setSortDirection: (direction: "asc" | "desc") => void;
	className?: string;
}) {
	return (
		<div
			className={cn(
				"mb-4 flex flex-wrap items-center justify-end gap-2 px-4 py-2 md:px-0",
				className,
			)}
		>
			<div className="flex items-center gap-3">
				<Select
					value={sortField}
					onValueChange={(value) => {
						if (value) setSortField(value as typeof sortField);
					}}
					items={[
						{ value: "publishedAtSort", label: "Published" },
						{ value: "createdAtSort", label: "Recorded" },
						{ value: "savedAtSort", label: "Saved" },
					]}
				>
					<SelectTrigger aria-label="Sort entries by">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectGroup>
							<SelectItem value="publishedAtSort">Published</SelectItem>
							<SelectItem value="createdAtSort">Recorded</SelectItem>
							<SelectItem value="savedAtSort">Saved</SelectItem>
						</SelectGroup>
					</SelectContent>
				</Select>
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
					<SelectTrigger aria-label="Sort entries">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectGroup>
							<SelectItem value="desc">Newest first</SelectItem>
							<SelectItem value="asc">Oldest first</SelectItem>
						</SelectGroup>
					</SelectContent>
				</Select>
				<div className="h-6 border-l border-border/70" aria-hidden="true" />
				<Link
					to="/record-link"
					aria-label="Save Link"
					className={buttonVariants({ variant: "ghost", size: "icon-sm" })}
				>
					<Plus />
				</Link>
			</div>
		</div>
	);
}
