import { ArrowDown, LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState } from "react";

const THRESHOLD = 80;

export function PullToRefresh({
	children,
	onRefresh,
}: {
	children: React.ReactNode;
	onRefresh: () => Promise<unknown>;
}) {
	const containerRef = useRef<HTMLDivElement>(null);
	const refreshRef = useRef(onRefresh);
	const refreshingRef = useRef(false);
	const [distance, setDistance] = useState(0);
	const [refreshing, setRefreshing] = useState(false);
	refreshRef.current = onRefresh;

	useEffect(() => {
		const container = containerRef.current;
		if (!container) return;
		let startX = 0;
		let startY = 0;
		let pulling = false;
		let currentDistance = 0;

		function reset() {
			pulling = false;
			currentDistance = 0;
			setDistance(0);
		}
		function onStart(event: TouchEvent) {
			if (
				event.touches.length !== 1 ||
				window.scrollY > 0 ||
				refreshingRef.current
			)
				return;
			const target = event.target as HTMLElement;
			for (
				let element: HTMLElement | null = target;
				element && element !== container;
				element = element.parentElement
			) {
				if (
					element.scrollHeight > element.clientHeight &&
					/(auto|scroll)/.test(getComputedStyle(element).overflowY)
				)
					return;
			}
			startX = event.touches[0].clientX;
			startY = event.touches[0].clientY;
			pulling = true;
		}
		function onMove(event: TouchEvent) {
			if (!pulling || event.touches.length !== 1 || window.scrollY > 0) return;
			const deltaY = event.touches[0].clientY - startY;
			const deltaX = event.touches[0].clientX - startX;
			if (deltaY <= 0 || Math.abs(deltaX) > deltaY) {
				reset();
				return;
			}
			if (event.cancelable) event.preventDefault();
			currentDistance = Math.min(deltaY * 0.5, 120);
			setDistance(currentDistance);
		}
		function onEnd() {
			if (!pulling) return;
			const shouldRefresh = currentDistance >= THRESHOLD;
			reset();
			if (shouldRefresh) {
				refreshingRef.current = true;
				setRefreshing(true);
				void Promise.resolve()
					.then(() => refreshRef.current())
					.finally(() => {
						refreshingRef.current = false;
						setRefreshing(false);
					})
					.catch(() => {});
			}
		}
		container.addEventListener("touchstart", onStart, { passive: true });
		container.addEventListener("touchmove", onMove, { passive: false });
		container.addEventListener("touchend", onEnd);
		container.addEventListener("touchcancel", reset);
		return () => {
			container.removeEventListener("touchstart", onStart);
			container.removeEventListener("touchmove", onMove);
			container.removeEventListener("touchend", onEnd);
			container.removeEventListener("touchcancel", reset);
		};
	}, []);

	return (
		<div ref={containerRef}>
			{(distance > 0 || refreshing) && (
				<output
					className="flex items-center justify-center gap-2 overflow-hidden text-sm text-muted-foreground"
					style={{ height: refreshing ? 48 : distance }}
				>
					{refreshing ? (
						<LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
					) : (
						<ArrowDown
							className={`size-4 transition-transform motion-reduce:transition-none ${distance >= THRESHOLD ? "rotate-180" : ""}`}
						/>
					)}
					{refreshing
						? "Refreshing..."
						: distance >= THRESHOLD
							? "Release to refresh"
							: "Pull to refresh"}
				</output>
			)}
			{children}
		</div>
	);
}
