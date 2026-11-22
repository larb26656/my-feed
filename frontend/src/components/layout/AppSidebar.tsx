import { Link, useRouterState } from "@tanstack/react-router";
import {
	Bookmark,
	Layers3,
	PanelLeftClose,
	PanelLeftOpen,
	RefreshCw,
	Rss,
	Settings,
	X,
} from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { SourceAvatar } from "#/components/SourceAvatar";
import { Button, buttonVariants } from "#/components/ui/button";
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

export function AppSidebar({
	collapsed = false,
	onToggle,
	onClose,
}: {
	collapsed?: boolean;
	onToggle?: () => void;
	onClose?: () => void;
}) {
	const { savedCount, savedCountError } = useFeedState();
	const { data: sidebarSources = [] } = useSidebarSources();
	const path = useRouterState({ select: (state) => state.location.pathname });
	return (
		<aside
			className={`flex h-full flex-col bg-card py-7 ${collapsed ? "px-2" : "px-5"}`}
		>
			<div
				className={`flex items-center ${collapsed ? "flex-col gap-3" : "justify-between"}`}
			>
				<Link
					to="/"
					onClick={onClose}
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
				{onToggle && (
					<Button
						variant="ghost"
						size="icon-sm"
						onClick={onToggle}
						aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
						title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
					>
						{collapsed ? <PanelLeftOpen /> : <PanelLeftClose />}
					</Button>
				)}
				{onClose && (
					<Button
						variant="ghost"
						size="icon-sm"
						onClick={onClose}
						aria-label="Close menu"
					>
						<X />
					</Button>
				)}
			</div>
			<p
				className={`${collapsed ? "sr-only" : ""} mt-14 px-3 text-[10px] font-bold tracking-widest text-muted-foreground`}
			>
				YOUR SPACE
			</p>
			<nav
				className={`mt-3 flex flex-col gap-1 ${collapsed ? "[&_a]:justify-center [&_a]:gap-0 [&_a]:!px-0" : ""}`}
				aria-label="Main navigation"
			>
				<NavItem
					onClick={(event) => {
						if (
							path === "/" &&
							event.button === 0 &&
							!event.metaKey &&
							!event.ctrlKey &&
							!event.shiftKey &&
							!event.altKey
						) {
							event.preventDefault();
							window.scrollTo({ top: 0, behavior: "smooth" });
						}
						onClose?.();
					}}
					to="/"
					active={path === "/"}
					activeOptions={{ exact: true }}
				>
					<Layers3 data-icon="inline-start" />
					<span className={collapsed ? "sr-only" : ""}>My Feed</span>
				</NavItem>
				<NavItem
					onClick={onClose}
					to="/fetch-jobs"
					active={path === "/fetch-jobs"}
				>
					<RefreshCw data-icon="inline-start" />
					<span className={collapsed ? "sr-only" : ""}>Fetch Jobs</span>
				</NavItem>
				<NavItem onClick={onClose} to="/saved" active={path === "/saved"}>
					<Bookmark data-icon="inline-start" />
					<span className={collapsed ? "sr-only" : ""}>Saved</span>
					{!collapsed &&
						(savedCountError ? (
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
						))}
				</NavItem>
				<NavItem
					onClick={onClose}
					to="/sources"
					active={path === "/sources"}
					activeOptions={{ exact: true }}
				>
					<Rss data-icon="inline-start" />
					<span className={collapsed ? "sr-only" : ""}>Manage Sources</span>
				</NavItem>
			</nav>
			<p
				className={`${collapsed ? "sr-only" : ""} mt-10 flex justify-between px-3 text-[10px] font-bold tracking-widest text-muted-foreground`}
			>
				YOUR SOURCES
			</p>
			<div
				className={`mt-3 flex min-h-0 flex-col overflow-y-auto ${collapsed ? "[&_a]:justify-center [&_a]:gap-0 [&_a]:!px-0" : ""}`}
			>
				{sidebarSources.map((source) => (
					<NavItem
						onClick={onClose}
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
						<span className={collapsed ? "sr-only" : "truncate"}>
							{source.name}
						</span>
					</NavItem>
				))}
			</div>
			<div
				className={`mt-auto flex flex-col gap-1 pt-4 ${collapsed ? "items-center [&_a]:justify-center [&_a]:gap-0 [&_a]:!px-0" : ""}`}
			>
				<ThemeToggle compact={collapsed} />
				<NavItem onClick={onClose} to="/settings" active={path === "/settings"}>
					<Settings data-icon="inline-start" />
					<span className={collapsed ? "sr-only" : ""}>Settings</span>
				</NavItem>
			</div>
		</aside>
	);
}
