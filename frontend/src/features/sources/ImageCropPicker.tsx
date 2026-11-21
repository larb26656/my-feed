import { Camera } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import Cropper, { type Area } from "react-easy-crop";
import { SourceAvatar } from "#/components/SourceAvatar";

export function useImageCrop() {
	const [image, setImage] = useState<string | null>(null);
	const [crop, setCrop] = useState({ x: 0, y: 0 });
	const [zoom, setZoom] = useState(1);
	const [area, setArea] = useState<Area | null>(null);
	const [error, setError] = useState("");
	useEffect(
		() => () => {
			if (image) URL.revokeObjectURL(image);
		},
		[image],
	);
	const onCropComplete = useCallback(
		(_: Area, pixels: Area) => setArea(pixels),
		[],
	);
	return {
		image,
		area,
		crop,
		zoom,
		error,
		setImage,
		setArea,
		setCrop,
		setZoom,
		setError,
		onCropComplete,
	};
}
type CropState = ReturnType<typeof useImageCrop>;
export async function prepareImage(image: string, area: Area): Promise<Blob> {
	const img = new Image();
	img.src = image;
	await img.decode();
	const canvas = document.createElement("canvas");
	canvas.width = 512;
	canvas.height = 512;
	const context = canvas.getContext("2d");
	if (!context) throw new Error("Your browser could not process this image.");
	context.drawImage(
		img,
		area.x,
		area.y,
		area.width,
		area.height,
		0,
		0,
		512,
		512,
	);
	const blob = await new Promise<Blob | null>((resolve) =>
		canvas.toBlob(resolve, "image/jpeg", 0.9),
	);
	if (!blob || blob.size > 2 * 1024 * 1024)
		throw new Error(
			"Could not prepare an image under 2 MB. Please choose another image.",
		);
	return blob;
}
export function ImageCropPicker({
	state,
	disabled = false,
	name = "Source",
	sourceId,
	imageId,
}: {
	state: CropState;
	disabled?: boolean;
	name?: string;
	sourceId?: string;
	imageId?: string | null;
}) {
	const {
		image,
		crop,
		zoom,
		setImage,
		setArea,
		setCrop,
		setZoom,
		setError,
		onCropComplete,
	} = state;
	return (
		<div className="grid gap-4">
			<label className="group mx-auto flex w-fit cursor-pointer items-center gap-4 has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-50">
				<input
					className="peer sr-only"
					aria-label="Choose image"
					disabled={disabled}
					type="file"
					accept="image/*"
					onChange={(e) => {
						const file = e.target.files?.[0];
						e.target.value = "";
						setError("");
						if (file) {
							const url = URL.createObjectURL(file);
							const probe = new Image();
							probe.onload = () => {
								setArea(null);
								setImage(url);
								setCrop({ x: 0, y: 0 });
								setZoom(1);
							};
							probe.onerror = () => {
								URL.revokeObjectURL(url);
								setArea(null);
								setImage(null);
								setError(
									"Your browser could not open this image. Please choose another.",
								);
							};
							probe.src = url;
						}
					}}
				/>
				<span className="relative shrink-0 rounded-full peer-focus-visible:ring-2 peer-focus-visible:ring-ring peer-focus-visible:ring-offset-2">
					{image ? (
						<img
							src={image}
							alt=""
							className="size-20 rounded-full object-cover"
						/>
					) : (
						<SourceAvatar
							name={name}
							sourceId={sourceId}
							imageId={imageId}
							size={80}
						/>
					)}
					<span className="absolute inset-0 flex items-center justify-center rounded-full bg-black/40 text-white transition-colors group-hover:bg-black/55">
						<Camera aria-hidden="true" className="size-6" />
					</span>
				</span>
			</label>
			{image && (
				<>
					<div className="relative h-72 overflow-hidden rounded-lg bg-muted">
						<Cropper
							image={image}
							crop={crop}
							zoom={zoom}
							aspect={1}
							onCropChange={setCrop}
							onZoomChange={setZoom}
							onCropComplete={onCropComplete}
						/>
					</div>
					<label className="grid gap-2">
						Zoom
						<input
							type="range"
							min="1"
							max="3"
							step="0.01"
							value={zoom}
							onChange={(e) => setZoom(Number(e.target.value))}
						/>
					</label>
				</>
			)}
		</div>
	);
}
