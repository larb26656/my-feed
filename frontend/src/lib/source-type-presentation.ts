import { Layers3, Rss } from "lucide-react";
import type { ComponentType, SVGProps } from "react";
import {
	BlueskyIcon,
	DevtoIcon,
	MediumIcon,
	YouTubeIcon,
} from "#/components/icons/source-types";

type SourceTypePresentation = {
	label: string;
	icon: ComponentType<SVGProps<SVGSVGElement>>;
};

const sourceTypePresentation: Record<string, SourceTypePresentation> = {
	rss: { label: "RSS", icon: Rss },
	manual: { label: "Manual", icon: Layers3 },
	youtube: { label: "YouTube", icon: YouTubeIcon },
	bluesky: { label: "Bluesky", icon: BlueskyIcon },
	medium: { label: "Medium", icon: MediumIcon },
	devto: { label: "DEV", icon: DevtoIcon },
};

export function getSourceTypePresentation(
	type: string,
): SourceTypePresentation {
	return Object.hasOwn(sourceTypePresentation, type)
		? sourceTypePresentation[type]
		: { label: type, icon: Layers3 };
}
