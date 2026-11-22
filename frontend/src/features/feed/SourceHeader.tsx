import { Link } from "@tanstack/react-router";
import { Settings2 } from "lucide-react";
import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import { buttonVariants } from "#/components/ui/button";
import { Skeleton } from "#/components/ui/skeleton";
import { cn } from "#/lib/utils";
import type { Source } from "./types";

function sourceDescription(source: Source) {
	const value = source.description;
	return typeof value === "string" ? value.trim() : "";
}

function ManageSourcesLink({ sourceId }: { sourceId?: string }) {
	return (
		<Link
			to="/sources"
			search={sourceId ? { sourceId } : {}}
			className={cn(
				buttonVariants(),
				"mt-5 bg-[#2563eb] px-5 text-white hover:bg-[#1d4ed8]",
			)}
		>
			<Settings2 data-icon="inline-start" />
			Manage sources
		</Link>
	);
}

function SourceBanner() {
	return (
		<div
			aria-hidden="true"
			className="absolute inset-0 bg-[linear-gradient(120deg,#1e3a8a_0%,#1d4ed8_46%,#2563eb_100%)] [--wave:#93c5fd] [--wave-deep:#1e40af] [--wave-soft:#dbeafe] dark:bg-[linear-gradient(120deg,#071525_0%,#0c2748_48%,#16467c_100%)] dark:[--wave:#3b82f6] dark:[--wave-deep:#12386a] dark:[--wave-soft:#60a5fa]"
		>
			<svg
				aria-hidden="true"
				className="h-full w-full"
				viewBox="0 0 1200 320"
				preserveAspectRatio="xMidYMid slice"
			>
				<path
					fill="var(--wave-deep)"
					fillOpacity="0.85"
					d="M0 70C180 130 320 20 520 62C760 112 900 20 1120 78C1160 90 1184 96 1200 92V0H0Z"
				/>
				<path
					fill="var(--wave)"
					fillOpacity="0.55"
					d="M0 150C160 90 300 210 520 168C760 122 920 70 1200 150V230C980 180 820 270 560 220C300 170 140 220 0 190Z"
				/>
				<path
					fill="var(--wave-soft)"
					fillOpacity="0.38"
					d="M0 210C220 160 380 270 640 230C920 186 1040 250 1200 214V320H0C40 280 80 250 0 210Z"
				/>
			</svg>
		</div>
	);
}

export function SourceHeader({
	source,
	loading,
	error,
}: {
	source?: Source;
	loading: boolean;
	error: boolean;
}) {
	const description = source ? sourceDescription(source) : "";

	return (
		<header className="border-b border-border [--avatar-ring:color-mix(in_oklch,var(--muted)_50%,var(--background))] md:mb-6 md:overflow-hidden md:rounded-xl md:border md:bg-card md:[--avatar-ring:var(--card)] dark:[--avatar-ring:var(--background)] md:dark:[--avatar-ring:var(--card)]">
			<div className="relative h-36 sm:h-40 md:h-44">
				<SourceBanner />
				<div className="absolute inset-x-0 bottom-0 z-10 flex translate-y-1/2 justify-center">
					{source ? (
						<SourceAvatar
							name={source.name}
							imageId={source.profileImageId}
							sourceId={source.id}
							size={104}
							className="bg-card shadow-[0_8px_20px_rgba(0,0,0,0.22),0_0_0_3px_var(--avatar-ring)]"
						/>
					) : (
						<Skeleton className="size-[104px] rounded-full shadow-[0_0_0_4px_var(--avatar-ring)]" />
					)}
				</div>
			</div>
			<div className="flex flex-col items-center px-4 pt-16 pb-6 text-center md:px-8 md:pt-[4.5rem] md:pb-8">
				{source ? (
					<>
						<h1 className="max-w-2xl text-2xl font-bold tracking-tight break-words md:text-3xl">
							{source.name}
						</h1>
						<div className="mt-3 flex flex-wrap items-center justify-center gap-2">
							<SourceTypeBadge type={source.type} />
							{!source.active && (
								<span className="text-xs text-muted-foreground">Inactive</span>
							)}
						</div>
						{description && (
							<p className="mt-3 max-w-md text-sm text-pretty text-muted-foreground">
								{description}
							</p>
						)}
						<ManageSourcesLink sourceId={source.id} />
					</>
				) : loading ? (
					<div className="flex flex-col items-center" aria-busy="true">
						<span className="sr-only">Loading source</span>
						<Skeleton className="h-7 w-40" />
						<Skeleton className="mt-3 h-5 w-20 rounded-full" />
					</div>
				) : (
					<div>
						<h1 className="text-2xl font-bold tracking-tight">Source</h1>
						<p className="mt-2 text-sm text-muted-foreground">
							{error
								? "Could not load source details."
								: "Source details are unavailable."}
						</p>
						<ManageSourcesLink />
					</div>
				)}
			</div>
		</header>
	);
}
