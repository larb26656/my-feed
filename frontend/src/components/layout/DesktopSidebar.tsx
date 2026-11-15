import { Link, useRouterState } from "@tanstack/react-router";
import { Bookmark, Layers3, RefreshCw, Settings } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { SourceAvatar } from "#/components/SourceAvatar";
import { buttonVariants } from "#/components/ui/button";
import { useFeedState } from "#/features/feed/FeedState";

type NavItemProps = Omit<
	ComponentProps<typeof Link>,
	"children" | "className"
> & {
	active: boolean;
	children: ReactNode;
	className?: string;
	variant?: "default" | "source";
};

function NavItem({
	active,
	children,
	className,
	variant = "default",
	...linkProps
}: NavItemProps) {
	const variantClass =
		variant === "source"
			? "h-auto min-h-10 justify-start px-3 text-left text-xs"
			: "h-11 w-full justify-start";

	return (
		<Link
			{...linkProps}
			className={buttonVariants({
				variant: active ? "secondary" : "ghost",
				className: `${variantClass} ${className ?? ""}`,
			})}
		>
			{children}
		</Link>
	);
}

export function DesktopSidebar() {
	const { sources, savedCount, savedCountError } = useFeedState();
	const path = useRouterState({ select: (state) => state.location.pathname });
	return (
		<aside className="sticky top-0 hidden h-screen flex-col border-r bg-card px-5 py-7 md:flex">
			<Link
				to="/"
				className="flex items-center gap-2 text-left text-xl font-bold tracking-tight"
			>
				<span className="grid size-9 place-items-center">
					<img
						src="/my-feed-mascot.png"
						alt=""
						className="size-9 object-contain"
					/>
				</span>
				My feed
			</Link>
			<p className="mt-14 px-3 text-[10px] font-bold tracking-widest text-muted-foreground">
				YOUR SPACE
			</p>
			<nav className="mt-3 flex flex-col gap-1" aria-label="Main navigation">
				<NavItem to="/" active={path === "/"} activeOptions={{ exact: true }}>
					<Layers3 data-icon="inline-start" />
					My Feed
				</NavItem>
				<NavItem to="/fetch-jobs" active={path === "/fetch-jobs"}>
					<RefreshCw data-icon="inline-start" />
					Fetch Jobs
				</NavItem>
				<NavItem to="/saved" active={path === "/saved"}>
					<Bookmark data-icon="inline-start" />
					Saved
					{savedCountError ? (
						<span
							className="ml-auto"
							role="alert"
							aria-label="Could not load saved count"
						>
							!
						</span>
					) : (
						savedCount !== undefined && (
							<span className="ml-auto">{savedCount}</span>
						)
					)}
				</NavItem>
			</nav>
			<p className="mt-10 flex justify-between px-3 text-[10px] font-bold tracking-widest text-muted-foreground">
				YOUR SOURCES <span>{sources.length}</span>
			</p>
			<div className="mt-3 flex min-h-0 flex-col overflow-y-auto">
				{sources.map((source) => (
					<NavItem
						key={source.id}
						to="/sources/$sourceId"
						params={{ sourceId: source.id } as never}
						active={path === `/sources/${source.id}`}
						variant="source"
					>
						<SourceAvatar name={source.name} image={source.image} size={24} />
						<span className="truncate">{source.name}</span>
					</NavItem>
				))}
				<NavItem
					to="/sources"
					active={path === "/sources"}
					className="mt-2 h-11"
					activeOptions={{ exact: true }}
				>
					Manage Sources
				</NavItem>
			</div>
			<NavItem to="/settings" active={path === "/settings"} className="mt-auto">
				<Settings data-icon="inline-start" />
				Settings
			</NavItem>
		</aside>
	);
}
