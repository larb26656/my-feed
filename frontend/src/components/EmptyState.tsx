import type { ReactNode } from "react";

export const EMPTY_STATE_ILLUSTRATIONS = {
	feed: {
		src: "/empty-states/feed-empty.png",
		alt: "Mascot holding a box of reading items",
	},
	sleeping: {
		src: "/empty-states/sleeping.png",
		alt: "Sleeping mascot",
	},
	saved: {
		src: "/empty-states/saved-items.png",
		alt: "Mascot holding a saved items bookmark",
	},
	search: {
		src: "/empty-states/search-empty.png",
		alt: "Mascot searching",
	},
	error: {
		src: "/empty-states/error.png",
		alt: "Mascot showing an error symbol",
	},
} as const;

export type EmptyStatePreset = keyof typeof EMPTY_STATE_ILLUSTRATIONS;

export type EmptyStateProps = {
	illustration?: EmptyStatePreset;
	imageSrc?: string;
	imageAlt?: string;
	title: string;
	description?: string;
	action?: ReactNode;
	role?: "status" | "alert";
	size?: "default" | "compact" | "screen";
};

export function EmptyState({
	illustration = "feed",
	imageSrc: customImageSrc,
	imageAlt,
	title,
	description,
	action,
	role = "status",
	size = "default",
}: EmptyStateProps) {
	const preset =
		illustration in EMPTY_STATE_ILLUSTRATIONS
			? EMPTY_STATE_ILLUSTRATIONS[illustration as EmptyStatePreset]
			: undefined;
	const imageSrc = customImageSrc ?? preset?.src;
	const resolvedImageAlt = imageAlt ?? preset?.alt ?? "";
	const sizeClasses = {
		default: "min-h-80 px-5 py-12 [&_img]:w-56",
		compact: "px-3 py-5 [&_img]:w-36",
		screen: "min-h-screen px-5 py-12 [&_img]:w-56",
	}[size];

	return (
		<section
			aria-live={role === "alert" ? "assertive" : "polite"}
			className={`flex flex-col items-center justify-center text-center ${sizeClasses}`}
			role={role}
		>
			<img
				src={imageSrc}
				alt={resolvedImageAlt}
				className="mb-5 h-auto w-56 max-w-full object-contain"
			/>
			<h2 className="text-lg font-semibold text-foreground">{title}</h2>
			{description && (
				<p className="mt-2 max-w-sm text-sm leading-relaxed text-muted-foreground">
					{description}
				</p>
			)}
			{action && <div className="mt-5">{action}</div>}
		</section>
	);
}
