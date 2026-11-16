import { Link, useLocation } from "@tanstack/react-router";
import { Bookmark, Layers3, Rss } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "#/lib/utils";

const destinations = [
	{ to: "/", label: "My Feed", Icon: Layers3 },
	{ to: "/saved", label: "Saved", Icon: Bookmark },
	{ to: "/sources", label: "Sources", Icon: Rss },
] as const;

export function MobileNavigation() {
	const pathname = useLocation({ select: (location) => location.pathname });
	// Show navigation again when the destination changes.
	return <FloatingNavigation key={pathname} pathname={pathname} />;
}

function FloatingNavigation({ pathname }: { pathname: string }) {
	const [visible, setVisible] = useState(true);

	useEffect(() => {
		const scrollPosition = () =>
			Math.max(
				0,
				Math.min(
					window.scrollY,
					Math.max(
						0,
						document.documentElement.scrollHeight - window.innerHeight,
					),
				),
			);
		let previous = scrollPosition();
		let travel = 0;
		const onScroll = () => {
			// Clamp elastic overscroll so bouncing at the bottom does not reveal the dock.
			const current = scrollPosition();
			const delta = current - previous;
			previous = current;
			if (current < 32) {
				setVisible(true);
				travel = 0;
				return;
			}
			if (!delta) return;
			travel = Math.sign(delta) === Math.sign(travel) ? travel + delta : delta;
			if (travel >= 24) setVisible(false);
			if (travel <= -6) setVisible(true);
		};
		window.addEventListener("scroll", onScroll, { passive: true });
		return () => window.removeEventListener("scroll", onScroll);
	}, []);

	return (
		<div
			className="pointer-events-none fixed inset-x-0 z-20 flex justify-center px-4 md:hidden"
			style={{ bottom: "calc(env(safe-area-inset-bottom, 0px) + 16px)" }}
		>
			<nav
				aria-label="Main navigation"
				onFocusCapture={() => setVisible(true)}
				className={cn(
					"pointer-events-auto flex items-center gap-1 rounded-full bg-card p-1.5 shadow-lg shadow-black/20 supports-[backdrop-filter]:bg-card/65 supports-[backdrop-filter]:backdrop-blur-xl",
					"transition-[transform,opacity] duration-200 ease-out motion-reduce:transition-none",
					!visible &&
						"pointer-events-none translate-y-[calc(100%+3rem)] opacity-0",
				)}
			>
				{destinations.map(({ to, label, Icon }) => {
					const active =
						to === "/"
							? pathname === "/"
							: pathname === to || pathname.startsWith(`${to}/`);
					return (
						<Link
							key={to}
							to={to}
							activeOptions={{ exact: to === "/" }}
							aria-label={label}
							aria-current={active ? "page" : undefined}
							className={cn(
								"flex min-h-12 min-w-12 items-center justify-center gap-2 rounded-full px-3 text-sm font-medium outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
								active
									? "bg-secondary/70 text-primary"
									: "text-muted-foreground",
							)}
						>
							<Icon className="size-5 shrink-0" aria-hidden="true" />
							{active && <span>{label}</span>}
						</Link>
					);
				})}
			</nav>
		</div>
	);
}
