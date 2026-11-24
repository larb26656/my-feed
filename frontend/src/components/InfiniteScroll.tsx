import { useEffect, useRef } from "react";
import { Button } from "#/components/ui/button";

type NextPage = {
	hasPage: boolean;
	isLoading: boolean;
	isError: boolean;
	load: () => Promise<unknown>;
};

export function useInfiniteScroll({
	hasPage,
	isLoading,
	isError,
	load,
	rootMargin = "200px",
}: NextPage & { rootMargin?: string }) {
	const sentinelRef = useRef<HTMLDivElement>(null);
	const pending = useRef(false);
	useEffect(() => {
		const node = sentinelRef.current;
		if (
			!node ||
			!hasPage ||
			isLoading ||
			isError ||
			typeof IntersectionObserver === "undefined"
		)
			return;
		let active = true;
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					!active ||
					pending.current ||
					!entries.some((entry) => entry.isIntersecting)
				)
					return;
				pending.current = true;
				void load()
					.catch(() => {})
					.finally(() => {
						pending.current = false;
					});
			},
			{ rootMargin },
		);
		observer.observe(node);
		return () => {
			active = false;
			observer.disconnect();
		};
	}, [hasPage, isLoading, isError, load, rootMargin]);
	return sentinelRef;
}

export function InfiniteScroll({
	children,
	next,
	rootMargin,
	loading,
	error,
}: {
	children: React.ReactNode;
	next: NextPage;
	rootMargin?: string;
	loading?: React.ReactNode;
	error?: React.ReactNode;
}) {
	const sentinelRef = useInfiniteScroll({ ...next, rootMargin });
	const unsupported = typeof IntersectionObserver === "undefined";
	return (
		<>
			{children}
			<div ref={sentinelRef} className="h-px" aria-hidden="true" />
			{next.isLoading &&
				(loading ?? (
					<output className="block py-4 text-center text-sm text-muted-foreground">
						Loading more...
					</output>
				))}
			{next.isError &&
				(error ?? (
					<div className="py-3 text-center">
						<p role="alert" className="text-sm text-destructive">
							Could not load more. Please try again.
						</p>
						<Button
							variant="outline"
							className="mt-2"
							onClick={() => void next.load()}
						>
							Try again
						</Button>
					</div>
				))}
			{unsupported && next.hasPage && !next.isLoading && !next.isError && (
				<div className="py-3 text-center">
					<Button variant="outline" onClick={() => void next.load()}>
						Load more
					</Button>
				</div>
			)}
		</>
	);
}
