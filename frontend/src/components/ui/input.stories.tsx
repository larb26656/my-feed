import type { Meta, StoryObj } from "@storybook/react-vite";
import { Input } from "./input";

const meta = {
	title: "UI/Input",
	component: Input,
	args: { "aria-label": "Source name", placeholder: "Source name" },
} satisfies Meta<typeof Input>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Default: Story = {};
export const Disabled: Story = {
	args: { disabled: true, value: "Unavailable" },
};
export const Invalid: Story = {
	args: { "aria-invalid": true, defaultValue: "Invalid source" },
};
