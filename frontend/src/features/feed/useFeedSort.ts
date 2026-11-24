import { useState } from "react";
import type { EntryFilters } from "./types";

export function useFeedSort(defaultSortField: EntryFilters["sortField"]) {
	const [sortDirection, setSortDirection] =
		useState<EntryFilters["sortDirection"]>("desc");
	const [chosenSortField, setSortField] = useState<
		EntryFilters["sortField"] | null
	>(null);
	return {
		sortField: chosenSortField ?? defaultSortField,
		setSortField,
		sortDirection,
		setSortDirection,
	};
}
