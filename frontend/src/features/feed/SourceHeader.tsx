import { SourceAvatar } from "#/components/SourceAvatar";
import { SourceTypeBadge } from "#/components/SourceTypeBadge";
import { Skeleton } from "#/components/ui/skeleton";
import {
	type SourceActionHandlers,
	SourceActions,
} from "#/features/sources/SourceActions";
import type { Source } from "./types";

function sourceDescription(source: Source) {
	const value = source.description;
	return typeof value === "string" ? value.trim() : "";
}

export function SourceHeader({
	source,
	loading,
	error,
	actions,
}: {
	source?: Source;
	loading: boolean;
	error: boolean;
	actions?: SourceActionHandlers;
}) {
	const description = source ? sourceDescription(source) : "";

	return (
		<header className="border-b border-border [--avatar-ring:color-mix(in_oklch,var(--muted)_50%,var(--background))] md:mb-3 md:overflow-hidden md:rounded-xl md:border md:bg-card md:[--avatar-ring:var(--card)] dark:[--avatar-ring:var(--background)] md:dark:[--avatar-ring:var(--card)]">
			<div className="relative h-24 bg-primary md:h-28">
				{source && actions && (
					<div className="absolute right-3 top-3 z-20">
						<SourceActions source={source} {...actions} />
					</div>
				)}
				<div className="absolute inset-x-0 bottom-0 z-10 flex translate-y-1/2 justify-center">
					{source ? (
						<SourceAvatar
							name={source.name}
							imageId={source.profileImageId}
							sourceId={source.id}
							size={80}
							className="bg-card shadow-[0_6px_16px_rgba(0,0,0,0.18),0_0_0_3px_var(--avatar-ring)]"
						/>
					) : (
						<Skeleton className="size-20 rounded-full shadow-[0_0_0_3px_var(--avatar-ring)]" />
					)}
				</div>
			</div>
			<div className="flex flex-col items-center px-4 pt-12 pb-4 text-center md:px-6 md:pb-5">
				{source ? (
					<>
						<h1 className="max-w-2xl text-2xl font-bold tracking-tight break-words">
							{source.name}
						</h1>
						<div className="mt-1.5 flex flex-wrap items-center justify-center gap-2">
							<SourceTypeBadge type={source.type} />
							{!source.active && (
								<span className="text-xs text-muted-foreground">Inactive</span>
							)}
						</div>
						{description && (
							<p className="mt-1.5 max-w-md text-sm text-pretty text-muted-foreground">
								{description}
							</p>
						)}
					</>
				) : loading ? (
					<div className="flex flex-col items-center" aria-busy="true">
						<span className="sr-only">Loading source</span>
						<Skeleton className="h-7 w-40" />
						<Skeleton className="mt-1.5 h-5 w-20 rounded-full" />
					</div>
				) : (
					<div>
						<h1 className="text-2xl font-bold tracking-tight">Source</h1>
						<p className="mt-1.5 text-sm text-muted-foreground">
							{error
								? "Could not load source details."
								: "Source details are unavailable."}
						</p>
					</div>
				)}
			</div>
		</header>
	);
}
