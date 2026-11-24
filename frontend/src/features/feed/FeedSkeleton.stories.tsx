import type { Meta, StoryObj } from "@storybook/react-vite";
import { FeedSkeleton } from "./FeedSkeleton";

const meta = {
	title: "Feed/FeedSkeleton",
	component: FeedSkeleton,
	decorators: [
		(Story) => (
			<div className="mx-auto max-w-3xl">
				<Story />
			</div>
		),
	],
} satisfies Meta<typeof FeedSkeleton>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Loading: Story = {};
