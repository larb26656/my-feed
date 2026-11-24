import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterContextProvider } from "@tanstack/react-router";
import type { ComponentProps } from "react";
import { getRouter } from "#/router";
import { FeedItem } from "./FeedItem";
import type { Entry, Source } from "./types";

const queryClient = new QueryClient({
	defaultOptions: { queries: { retry: false } },
});
const router = getRouter();
const entry: Entry = {
	id: "entry-1",
	sourceId: "source-1",
	title: "A better way to keep up with what you read",
	link: "https://example.com/article",
	description:
		"Discover ideas, save interesting articles, and catch up on your favorite sources at your own pace.",
	publishedAt: "2025-01-15T10:00:00Z",
	createdAt: "2025-01-15T10:00:00Z",
};
const source: Source = {
	id: "source-1",
	name: "The Reading List",
	type: "rss",
	active: true,
};

function FeedItemStory(
	props: Extract<ComponentProps<typeof FeedItem>, { disabled: boolean }>,
) {
	return <FeedItem {...props} />;
}

const meta = {
	title: "Feed/FeedItem",
	component: FeedItemStory,
	decorators: [
		(Story) => (
			<QueryClientProvider client={queryClient}>
				<RouterContextProvider router={router}>
					<div className="mx-auto max-w-3xl">
						<Story />
					</div>
				</RouterContextProvider>
			</QueryClientProvider>
		),
	],
	args: { entry, source, disabled: false, onToggle: () => {} },
} satisfies Meta<typeof FeedItemStory>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Default: Story = {};
export const WithThumbnail: Story = {
	args: { entry: { ...entry, thumbnail: "/empty-states/feed-empty.png" } },
};
export const ReadAndSaved: Story = {
	args: {
		entry: {
			...entry,
			readAt: "2025-01-16T10:00:00Z",
			savedAt: "2025-01-16T10:00:00Z",
		},
	},
};
export const Disabled: Story = { args: { disabled: true } };
