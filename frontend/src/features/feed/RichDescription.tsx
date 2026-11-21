import { type ReactNode, useEffect, useRef, useState } from "react";
import { cn } from "#/lib/utils";

const allowedTags = new Set([
	"A",
	"B",
	"BLOCKQUOTE",
	"BR",
	"CODE",
	"EM",
	"I",
	"LI",
	"OL",
	"P",
	"PRE",
	"S",
	"STRONG",
	"U",
	"UL",
]);
const blockedTags = new Set(["IFRAME", "OBJECT", "SCRIPT", "STYLE", "SVG"]);

function safeHref(value: string | null) {
	if (!value) return undefined;
	try {
		const url = new URL(value, window.location.origin);
		return url.protocol === "http:" || url.protocol === "https:"
			? url.href
			: undefined;
	} catch {
		return undefined;
	}
}

function renderNode(node: Node, key: number): ReactNode {
	if (node.nodeType === Node.TEXT_NODE) return node.textContent;
	if (!(node instanceof Element)) return null;
	if (blockedTags.has(node.tagName)) return null;

	if (node.tagName === "BR") return <br key={key} />;

	const children = Array.from(node.childNodes).map((child, index) =>
		renderNode(child, index),
	);
	if (!allowedTags.has(node.tagName)) return children;
	if (node.tagName === "A") {
		const href = safeHref(node.getAttribute("href"));
		return href ? (
			<a
				key={key}
				href={href}
				target="_blank"
				rel="noreferrer"
				className="text-primary underline underline-offset-2"
			>
				{children}
			</a>
		) : (
			<span key={key}>{children}</span>
		);
	}

	const Tag = node.tagName.toLowerCase() as
		| "b"
		| "blockquote"
		| "code"
		| "em"
		| "i"
		| "li"
		| "ol"
		| "p"
		| "pre"
		| "s"
		| "strong"
		| "u"
		| "ul";
	return <Tag key={key}>{children}</Tag>;
}

export function RichDescription({ html }: { html: string }) {
	const parsed = new DOMParser().parseFromString(html, "text/html");
	const contentRef = useRef<HTMLDivElement>(null);
	const [expanded, setExpanded] = useState(false);
	const [overflowing, setOverflowing] = useState(false);

	// biome-ignore lint/correctness/useExhaustiveDependencies: remeasure when html content changes
	useEffect(() => {
		const content = contentRef.current;
		if (!content) return;
		const measure = () => {
			const limit = parseFloat(getComputedStyle(content).lineHeight) * 5;
			setOverflowing(content.scrollHeight > limit + 1);
		};
		measure();
		if (typeof ResizeObserver === "undefined") return;
		const observer = new ResizeObserver(measure);
		observer.observe(content);
		return () => observer.disconnect();
	}, [html]);

	return (
		<div className="mt-1 text-sm leading-relaxed text-foreground/90">
			<div
				ref={contentRef}
				className={cn(
					"space-y-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_ul]:list-disc [&_ul]:pl-5",
					!expanded && "max-h-[7.5em] overflow-hidden",
					overflowing &&
						!expanded &&
						"[mask-image:linear-gradient(to_bottom,black_65%,transparent_100%)]",
				)}
				inert={overflowing && !expanded ? true : undefined}
			>
				{Array.from(parsed.body.childNodes).map((node, index) =>
					renderNode(node, index),
				)}
			</div>
			{overflowing && (
				<button
					type="button"
					aria-expanded={expanded}
					onClick={() => setExpanded(!expanded)}
					className="mt-1 font-medium text-muted-foreground hover:text-foreground hover:underline focus-visible:outline-2 focus-visible:outline-offset-2"
				>
					{expanded ? "See less" : "See more"}
				</button>
			)}
		</div>
	);
}
