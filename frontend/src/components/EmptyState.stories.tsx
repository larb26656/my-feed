import type { Meta, StoryObj } from "@storybook/react-vite";
import { EmptyState } from "./EmptyState";
import { Button } from "./ui/button";

const meta = {
	title: "Components/EmptyState",
	component: EmptyState,
	args: {
		title: "Your feed is empty",
		description: "Add a source to see items here.",
	},
} satisfies Meta<typeof EmptyState>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Feed: Story = { args: { action: <Button>Add source</Button> } };
export const Saved: Story = {
	args: {
		illustration: "saved",
		title: "No saved items yet",
		description: "Save an item to find it here later.",
	},
};
export const Search: Story = {
	args: {
		illustration: "search",
		title: "No results found",
		description: "Try a different search.",
		size: "compact",
	},
};
