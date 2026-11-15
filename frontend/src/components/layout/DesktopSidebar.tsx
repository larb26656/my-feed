import { Link, useRouterState } from "@tanstack/react-router";
import { Bookmark, Layers3, Rss, Settings } from "lucide-react";
import { buttonVariants } from "#/components/ui/button";
import { useFeedState } from "#/features/feed/FeedState";

export function DesktopSidebar() {
	const { sources, savedCount } = useFeedState();
	const path = useRouterState({ select: (state) => state.location.pathname });
	const navClass = (active: boolean, className: string) =>
		buttonVariants({ variant: active ? "secondary" : "ghost", className });
	return (
		<aside className="sticky top-0 hidden h-screen flex-col border-r bg-card px-5 py-7 md:flex">
			<Link
				to="/"
				className="flex items-center gap-2 text-left text-xl font-bold tracking-tight"
			>
				<span className="grid size-9 place-items-center rounded-xl bg-primary text-primary-foreground">
					<Rss size={20} strokeWidth={3} />
				</span>
				quick.feed
			</Link>
			<p className="mt-14 px-3 text-[10px] font-bold tracking-widest text-muted-foreground">
				YOUR SPACE
			</p>
			<nav className="mt-3 flex flex-col gap-1" aria-label="เมนูหลัก">
				<Link
					to="/"
					className={navClass(path === "/", "h-11 justify-start")}
					activeOptions={{ exact: true }}
				>
					<Layers3 data-icon="inline-start" />
					ฟีดของฉัน
				</Link>
				<Link
					to="/saved"
					className={navClass(path === "/saved", "h-11 justify-start")}
				>
					<Bookmark data-icon="inline-start" />
					บันทึกไว้
					{savedCount !== undefined && (
						<span className="ml-auto">{savedCount}</span>
					)}
				</Link>
			</nav>
			<p className="mt-10 flex justify-between px-3 text-[10px] font-bold tracking-widest text-muted-foreground">
				YOUR SOURCES <span>{sources.length}</span>
			</p>
			<div className="mt-3 flex min-h-0 flex-col overflow-y-auto">
				{sources.map((source) => (
					<Link
						key={source.id}
						to="/sources/$sourceId"
						params={{ sourceId: source.id }}
						className={navClass(
							path === `/sources/${source.id}`,
							"h-auto min-h-10 justify-start px-3 text-left text-xs",
						)}
					>
						<span className="truncate">{source.name}</span>
					</Link>
				))}
				<Link
					to="/sources"
					activeOptions={{ exact: true }}
					className={navClass(
						path === "/sources",
						"mt-2 justify-start text-xs",
					)}
				>
					จัดการ Sources
				</Link>
			</div>
			<Link
				to="/settings"
				className={navClass(path === "/settings", "mt-auto h-11 justify-start")}
			>
				<Settings data-icon="inline-start" />
				Settings
			</Link>
		</aside>
	);
}
