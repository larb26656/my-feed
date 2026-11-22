import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { Link, useLocation } from "@tanstack/react-router";
import { Menu } from "lucide-react";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import { AppSidebar } from "./AppSidebar";

export function PageHeader() {
	const [open, setOpen] = useState(false);
	const pathname = useLocation({ select: (location) => location.pathname });
	return (
		<header className="flex h-16 items-center gap-2 border-b bg-card px-4 md:hidden">
			<DialogPrimitive.Root open={open} onOpenChange={setOpen}>
				<DialogPrimitive.Trigger
					render={<Button variant="ghost" size="icon" aria-label="Open menu" />}
				>
					<Menu />
				</DialogPrimitive.Trigger>
				<DialogPrimitive.Portal>
					<DialogPrimitive.Backdrop className="fixed inset-0 z-50 bg-black/40 transition-opacity duration-200 ease-out data-starting-style:opacity-0 data-ending-style:opacity-0 motion-reduce:transition-none" />
					<DialogPrimitive.Popup className="fixed inset-y-0 left-0 z-50 w-[min(280px,85vw)] bg-card shadow-xl outline-none transition-transform duration-300 ease-out data-starting-style:-translate-x-full data-ending-style:-translate-x-full data-ending-style:duration-200 motion-reduce:transition-none">
						<DialogPrimitive.Title className="sr-only">
							Navigation menu
						</DialogPrimitive.Title>
						<AppSidebar key={pathname} onClose={() => setOpen(false)} />
					</DialogPrimitive.Popup>
				</DialogPrimitive.Portal>
			</DialogPrimitive.Root>
			<Link
				to="/"
				aria-label="My feed"
				className="grid size-8 place-items-center"
			>
				<img
					src="/my-feed-logo.png"
					alt=""
					className="size-8 object-contain dark:hidden"
				/>
				<img
					src="/my-feed-logo-dark.png"
					alt=""
					className="hidden size-8 object-contain dark:block"
				/>
			</Link>
		</header>
	);
}
