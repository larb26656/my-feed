import { Link, useRouterState } from "@tanstack/react-router";
import { Bookmark, Layers3, RefreshCw, Rss, Settings } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { SourceAvatar } from "#/components/SourceAvatar";
import { buttonVariants } from "#/components/ui/button";
import { useFeedState } from "#/features/feed/FeedState";
import { useSidebarSources } from "#/hooks/queries";
import { ThemeToggle } from "./ThemeToggle";

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
	const { savedCount, savedCountError } = useFeedState();
	const { data: sidebarSources = [] } = useSidebarSources();
	const path = useRouterState({ select: (state) => state.location.pathname });
	return (
		<aside className="sticky top-0 hidden h-screen flex-col border-r bg-card px-5 py-7 md:flex">
			<Link
				to="/"
				aria-label="My feed"
				className="flex items-center gap-2 text-left text-xl font-bold tracking-tight"
			>
				<span className="grid size-9 place-items-center">
					<img
						src="/my-feed-logo.png"
						alt=""
						className="size-9 object-contain dark:hidden"
					/>
					<img
						src="/my-feed-logo-dark.png"
						alt=""
						className="hidden size-9 object-contain dark:block"
					/>
				</span>
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
				<NavItem
					to="/sources"
					active={path === "/sources"}
					activeOptions={{ exact: true }}
				>
					<Rss data-icon="inline-start" />
					Manage Sources
				</NavItem>
			</nav>
			<p className="mt-10 flex justify-between px-3 text-[10px] font-bold tracking-widest text-muted-foreground">
				YOUR SOURCES
			</p>
			<div className="mt-3 flex min-h-0 flex-col overflow-y-auto">
				{sidebarSources.map((source) => (
					<NavItem
						key={source.id}
						to="/sources/$sourceId"
						params={{ sourceId: source.id } as never}
						active={path === `/sources/${source.id}`}
						variant="source"
					>
						<SourceAvatar
							name={source.name}
							imageId={source.profileImageId}
							sourceId={source.id}
							size={24}
						/>
						<span className="truncate">{source.name}</span>
					</NavItem>
				))}
			</div>
			<div className="mt-auto flex flex-col gap-1 pt-4">
				<ThemeToggle />
				<NavItem to="/settings" active={path === "/settings"}>
					<Settings data-icon="inline-start" />
					Settings
				</NavItem>
			</div>
		</aside>
	);
}
