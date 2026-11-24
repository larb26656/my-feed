import type { Meta, StoryObj } from "@storybook/react-vite";
import { ErrorState } from "./ErrorState";
import { Button } from "./ui/button";

const meta = {
	title: "Components/ErrorState",
	component: ErrorState,
	args: { error: "Please try again later." },
} satisfies Meta<typeof ErrorState>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Default: Story = {};
export const WithRetry: Story = {
	args: { title: "Unable to load your feed", action: <Button>Retry</Button> },
};
