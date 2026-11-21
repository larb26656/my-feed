import { Avatar, Style } from "@dicebear/core";
import definition from "@dicebear/styles/loops.json" with { type: "json" };
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { fetchImage } from "#/lib/api/client";
import { cn } from "#/lib/utils";

const loops = new Style(definition);

export function SourceAvatar({
	name,
	imageId,
	sourceId,
	size = 40,
	className,
}: {
	name: string;
	imageId?: string | null;
	sourceId?: string;
	size?: number;
	className?: string;
}) {
	const { data: blob } = useQuery({
		queryKey: ["source-profile-image", sourceId, imageId],
		queryFn: () => fetchImage(`/sources/${sourceId}/profile-image`),
		enabled: Boolean(sourceId && imageId),
		retry: false,
	});
	const [image, setImage] = useState<{ blob: Blob; url: string } | null>(null);
	useEffect(() => {
		if (!blob) {
			setImage(null);
			return;
		}
		const url = URL.createObjectURL(blob);
		setImage({ blob, url });
		return () => URL.revokeObjectURL(url);
	}, [blob]);
	const [failedImage, setFailedImage] = useState<string | null>(null);
	const seed = name.trim() || "Source";
	const sourceImage =
		image && image.blob === blob && failedImage !== image.url
			? image.url
			: undefined;
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
