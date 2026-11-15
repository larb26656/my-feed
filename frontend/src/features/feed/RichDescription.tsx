import type { ReactNode } from "react";

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
		return url.protocol === "http:" || url.protocol === "https:" ? url.href : undefined;
	} catch {
		return undefined;
	}
}

function renderNode(node: Node, key: number): ReactNode {
	if (node.nodeType === Node.TEXT_NODE) return node.textContent;
	if (!(node instanceof Element)) return null;
	if (blockedTags.has(node.tagName)) return null;

	const children = Array.from(node.childNodes).map((child, index) =>
		renderNode(child, index),
	);
	if (!allowedTags.has(node.tagName)) return children;
	if (node.tagName === "A") {
		const href = safeHref(node.getAttribute("href"));
		return href ? (
			<a key={key} href={href} target="_blank" rel="noreferrer" className="text-primary underline underline-offset-2">
				{children}
			</a>
		) : <span key={key}>{children}</span>;
	}

	const Tag = node.tagName.toLowerCase() as
		| "b" | "blockquote" | "br" | "code" | "em" | "i" | "li"
		| "ol" | "p" | "pre" | "s" | "strong" | "u" | "ul";
	return <Tag key={key}>{children}</Tag>;
}

export function RichDescription({ html }: { html: string }) {
	const parsed = new DOMParser().parseFromString(html, "text/html");
	return (
		<div className="mt-1 space-y-2 text-sm leading-relaxed text-foreground/90 [&_ol]:list-decimal [&_ol]:pl-5 [&_ul]:list-disc [&_ul]:pl-5">
			{Array.from(parsed.body.childNodes).map((node, index) => renderNode(node, index))}
		</div>
	);
}
