import { okJson } from "#/lib/api/test-helpers";

// Give older dialog lifecycle tests the same discovery response as the server.
export function withSourceTypes<T extends (...args: never[]) => unknown>(
	mock: T,
): T {
	return new Proxy(mock, {
		apply(target, receiver, args) {
			if (String(args[0]).includes("/source-types"))
				return sourceTypesResponse();
			return Reflect.apply(target, receiver, args);
		},
	});
}

export function sourceTypesResponse() {
	return Promise.resolve(
		okJson({
			types: [
				{
					code: "rss",
					label: "RSS feed",
					schema: {
						$schema: "https://json-schema.org/draft/2020-12/schema",
						type: "object",
						properties: {
							url: {
								type: "string",
								pattern:
									"^\\s*https?://([^\\s/@]+@)?(\\[[0-9a-fA-F:.]+\\]|[^\\s/:?#@\\[\\]]+)(:[0-9]+)?([/?#][^\\s]*)?\\s*$",
							},
						},
						required: ["url"],
						additionalProperties: false,
					},
					ui: {
						url: {
							label: "RSS / Atom feed URL",
							widget: "url",
							error: "Please enter a valid RSS / Atom URL",
						},
					},
				},
				{
					code: "youtube",
					label: "YouTube",
					schema: {
						$schema: "https://json-schema.org/draft/2020-12/schema",
						type: "object",
						additionalProperties: false,
						properties: {
							channelId: {
								type: "string",
								pattern:
									"^\\s*(https://www\\.youtube\\.com/channel/)?UC[A-Za-z0-9_-]{22}\\s*$",
							},
						},
						required: ["channelId"],
					},
					ui: {
						channelId: {
							label: "YouTube channel ID or URL",
							widget: "text",
							description: "Handles and custom URLs are not supported.",
							error:
								"Enter a channel ID starting with UC or a canonical YouTube channel URL",
						},
					},
				},
				{
					code: "bluesky",
					label: "Bluesky",
					schema: {
						$schema: "https://json-schema.org/draft/2020-12/schema",
						type: "object",
						additionalProperties: false,
						properties: {
							did: {
								type: "string",
								minLength: 1,
							},
						},
						required: ["did"],
					},
					ui: {
						did: {
							label: "Bluesky handle or profile URL",
							widget: "text",
							description:
								"Enter a handle like @alice.bsky.social or a bsky.app profile URL.",
							error: "Enter a Bluesky handle or profile URL",
						},
					},
				},
			],
		}),
	);
}
