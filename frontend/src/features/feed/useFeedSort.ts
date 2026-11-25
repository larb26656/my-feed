import { useState } from "react";
import type { EntryFilters } from "./types";
export function useFeedSort(sortField: EntryFilters["sortField"]) {
	const [sortDirection, setSortDirection] =
		useState<EntryFilters["sortDirection"]>("desc");
	return { sortField, sortDirection, setSortDirection };
}
