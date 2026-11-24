import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SourceAvatar } from "./SourceAvatar";

const queryClient = new QueryClient({
	defaultOptions: { queries: { retry: false } },
});
const meta = {
	title: "Components/SourceAvatar",
	component: SourceAvatar,
	decorators: [
		(Story) => (
			<QueryClientProvider client={queryClient}>
				<Story />
			</QueryClientProvider>
		),
	],
	args: { name: "My Feed" },
} satisfies Meta<typeof SourceAvatar>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Generated: Story = {};
export const Large: Story = { args: { name: "Reading list", size: 80 } };
export const EmptyName: Story = { args: { name: "" } };
