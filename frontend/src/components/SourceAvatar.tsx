import { Avatar, Style } from "@dicebear/core";
import definition from "@dicebear/styles/loops.json" with { type: "json" };
import { useMemo, useState } from "react";
import { cn } from "#/lib/utils";

const loops = new Style(definition);

export function SourceAvatar({
	name,
	image,
	size = 40,
	className,
}: {
	name: string;
	image?: string | null;
	size?: number;
	className?: string;
}) {
	const [failedImage, setFailedImage] = useState<string | null>(null);
	const seed = name.trim() || "Source";
	const sourceImage = image && failedImage !== image ? image : undefined;
	const generatedImage = useMemo(
		() =>
			new Avatar(loops, {
				seed,
				size: size * 2,
				borderRadius: 50,
			}).toDataUri(),
		[seed, size],
	);

	return (
		<img
			src={sourceImage ?? generatedImage}
			alt={`${seed} avatar`}
			width={size}
			height={size}
			onError={sourceImage ? () => setFailedImage(sourceImage) : undefined}
			className={cn(
				"shrink-0 overflow-hidden rounded-full object-cover",
				className,
			)}
			style={{ width: size, height: size }}
		/>
	);
}
