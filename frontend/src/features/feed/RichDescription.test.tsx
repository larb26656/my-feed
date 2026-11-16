// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RichDescription } from "./RichDescription";

describe("RichDescription", () => {
	it("displays formatting, literal text and origin links while retaining restrictions", () => {
		render(
			<RichDescription
				html={
					'<p>Read <strong>this</strong> &lt;script&gt; literally. <a href="https://example.test/docs/guide">Guide</a><script>hidden</script><img src="https://example.test/image.png"></p>'
				}
			/>,
		);
		expect(screen.getByText("this").tagName).toBe("STRONG");
		expect(screen.getByText(/<script> literally/)).toBeTruthy();
		expect(
			screen.getByRole("link", { name: "Guide" }).getAttribute("href"),
		).toBe("https://example.test/docs/guide");
		expect(screen.queryByText("hidden")).toBeNull();
		expect(screen.queryByRole("img")).toBeNull();
	});
});
