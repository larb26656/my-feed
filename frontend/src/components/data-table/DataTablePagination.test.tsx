// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { DataTablePagination } from "./DataTablePagination";

afterEach(() => {
	cleanup();
});

test("renders numbered pages and jumps on click for page mode", async () => {
	const onPageChange = vi.fn();
	const user = userEvent.setup();

	render(
		<DataTablePagination
			pagination={{
				mode: "page",
				pageIndex: 0,
				pageSize: 10,
				pageCount: 5,
				onPageChange,
				onPageSizeChange: vi.fn(),
			}}
			canPreviousPage={false}
			canNextPage={true}
			onFirstPage={vi.fn()}
			onPreviousPage={vi.fn()}
			onNextPage={vi.fn()}
			onLastPage={vi.fn()}
		/>,
	);

	expect(screen.getByText("Page 1 of 5")).toBeTruthy();
	expect(
		screen.getByRole("navigation", { name: "Pagination pages" }),
	).toBeTruthy();
	await user.click(screen.getByRole("button", { name: "Page 3" }));
	expect(onPageChange).toHaveBeenCalledWith(2);
});

test("collapses long page ranges with ellipsis", () => {
	render(
		<DataTablePagination
			pagination={{
				mode: "page",
				pageIndex: 4,
				pageSize: 10,
				pageCount: 12,
				onPageChange: vi.fn(),
				onPageSizeChange: vi.fn(),
			}}
			canPreviousPage={true}
			canNextPage={true}
			onFirstPage={vi.fn()}
			onPreviousPage={vi.fn()}
			onNextPage={vi.fn()}
			onLastPage={vi.fn()}
		/>,
	);

	expect(screen.getByRole("button", { name: "Page 1" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Page 5" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Page 12" })).toBeTruthy();
	expect(screen.queryByRole("button", { name: "Page 2" })).toBeNull();
	expect(screen.getAllByText("…").length).toBeGreaterThan(0);
});

test("hides numbered pages in cursor mode", () => {
	render(
		<DataTablePagination
			pagination={{
				mode: "cursor",
				pageIndex: 0,
				pageSize: 10,
				hasPreviousPage: false,
				hasNextPage: true,
				onPreviousPage: vi.fn(),
				onNextPage: vi.fn(),
				onPageSizeChange: vi.fn(),
			}}
			canPreviousPage={false}
			canNextPage={true}
			onFirstPage={vi.fn()}
			onPreviousPage={vi.fn()}
			onNextPage={vi.fn()}
			onLastPage={vi.fn()}
		/>,
	);

	expect(screen.getByText("Page 1")).toBeTruthy();
	expect(
		screen.queryByRole("navigation", { name: "Pagination pages" }),
	).toBeNull();
	expect(screen.getByRole("button", { name: "Previous page" })).toBeTruthy();
	expect(screen.getByRole("button", { name: "Next page" })).toBeTruthy();
});
