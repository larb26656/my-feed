// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { failJson, okJson } from "#/lib/api/test-helpers";
import { SourcesPage } from "./SourcesPage";

vi.mock("@tanstack/react-router", () => ({
	Link: ({
		children,
		...props
	}: ComponentProps<"a"> & { to?: string; params?: unknown }) => (
		<a href="/sources" {...props}>
			{children}
		</a>
	),
}));

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});

const mediumDefinition = {
	code: "medium",
	label: "Medium",
	fieldOrder: ["kind", "identifier"],
	schema: {
		$schema: "https://json-schema.org/draft/2020-12/schema",
		type: "object",
		additionalProperties: false,
		properties: {
			identifier: { type: "string" },
			kind: {
				type: "string",
				enum: ["profile", "publication"],
			},
		},
		required: ["kind", "identifier"],
		allOf: [
			{
				if: {
					properties: { kind: { const: "profile" } },
					required: ["kind"],
				},
				// JSON Schema Draft 2020-12 conditional keyword
				// biome-ignore lint/suspicious/noThenProperty: schema keyword
				then: {
					properties: {
						identifier: {
							type: "string",
							pattern: "^\\s*@?[A-Za-z0-9_-]+\\s*$",
						},
					},
				},
			},
			{
				if: {
					properties: { kind: { const: "publication" } },
					required: ["kind"],
				},
				// biome-ignore lint/suspicious/noThenProperty: schema keyword
				then: {
					properties: {
						identifier: {
							type: "string",
							pattern: "^\\s*[A-Za-z0-9_-]+\\s*$",
						},
					},
				},
			},
		],
	},
	ui: {
		kind: {
			label: "Kind",
			widget: "select",
		},
		identifier: {
			label: "Identifier",
			widget: "text",
			description:
				"Profile username like alice or @alice, or a bare publication slug.",
			error: "Enter a Medium identifier without a URL or path",
		},
	},
};

function mediumTypesResponse() {
	return Promise.resolve(okJson({ types: [mediumDefinition] }));
}

function isSourcesList(url: string, init?: RequestInit) {
	if (init?.method && init.method !== "GET") return false;
	const path = url.replace(/\?.*$/, "");
	return path === "/api/v1/sources" || path === "/api/v1/sources/";
}

function emptyList(url: string, init?: RequestInit) {
	if (isSourcesList(url, init)) {
		return okJson([], { page: 1, pageSize: 10, total: 0, totalPages: 1 });
	}
	return okJson({
		id: "md",
		name: "Alice",
		type: "medium",
		config: { kind: "profile", identifier: "alice" },
		active: true,
	});
}

function withMediumTypes<T extends (...args: never[]) => unknown>(mock: T): T {
	return new Proxy(mock, {
		apply(target, receiver, args) {
			if (String(args[0]).includes("/source-types"))
				return mediumTypesResponse();
			return Reflect.apply(target, receiver, args);
		},
	});
}

function renderPage() {
	return render(
		<QueryClientProvider
			client={
				new QueryClient({ defaultOptions: { queries: { retry: false } } })
			}
		>
			<SourcesPage />
		</QueryClientProvider>,
	);
}

async function openMediumAdd(user: ReturnType<typeof userEvent.setup>) {
	await user.click(await screen.findByRole("button", { name: "Add Source" }));
	await user.click(screen.getByLabelText("Medium"));
}

test("adds a Medium Profile Source from a username", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			return okJson({
				id: "md",
				name: "Alice",
				type: "medium",
				config: { kind: "profile", identifier: "alice" },
				active: true,
			});
		}
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	expect(
		screen
			.getByLabelText("Kind")
			.compareDocumentPosition(screen.getByLabelText("Identifier")) &
			Node.DOCUMENT_POSITION_FOLLOWING,
	).toBeTruthy();
	await user.selectOptions(screen.getByLabelText("Kind"), "profile");
	await user.type(screen.getByLabelText("Source name"), "Alice");
	await user.type(screen.getByLabelText("Identifier"), "@Alice");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "POST"),
		).toBe(true),
	);
	const postCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "POST",
	);
	expect(JSON.parse(String(postCall?.[1]?.body))).toEqual({
		name: "Alice",
		type: "medium",
		config: { kind: "profile", identifier: "@Alice" },
	});
});

test("adds a Medium Publication Source from a slug", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (init?.method === "POST") {
			return okJson({
				id: "md-pub",
				name: "Alice Pub",
				type: "medium",
				config: { kind: "publication", identifier: "alice" },
				active: true,
			});
		}
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	await user.selectOptions(screen.getByLabelText("Kind"), "publication");
	await user.type(screen.getByLabelText("Source name"), "Alice Pub");
	await user.type(screen.getByLabelText("Identifier"), "alice");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "POST"),
		).toBe(true),
	);
	const postCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "POST",
	);
	expect(JSON.parse(String(postCall?.[1]?.body))).toEqual({
		name: "Alice Pub",
		type: "medium",
		config: { kind: "publication", identifier: "alice" },
	});
});

test("rejects a Medium identifier URL before save", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) =>
		emptyList(url, init),
	);
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	await user.selectOptions(screen.getByLabelText("Kind"), "profile");
	await user.type(screen.getByLabelText("Source name"), "Alice");
	await user.type(
		screen.getByLabelText("Identifier"),
		"https://medium.com/@alice",
	);
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(
		await screen.findByText("Enter a Medium identifier without a URL or path"),
	).toBeTruthy();
	expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
		false,
	);
});

test("rejects a Medium Publication leading @ before save", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) =>
		emptyList(url, init),
	);
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	await user.selectOptions(screen.getByLabelText("Kind"), "publication");
	await user.type(screen.getByLabelText("Source name"), "Alice Pub");
	await user.type(screen.getByLabelText("Identifier"), "@alice");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	expect(
		await screen.findByText("Enter a Medium identifier without a URL or path"),
	).toBeTruthy();
	expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(
		false,
	);
});

test("previews a Medium Profile and clears on identifier edit", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return okJson({
				entry: {
					title: "Story",
					link: "https://medium.com/@alice/story",
					description: "Body",
				},
			});
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	await user.selectOptions(screen.getByLabelText("Kind"), "profile");
	const input = screen.getByLabelText("Identifier");
	await user.type(input, "alice");
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	const link = await screen.findByRole("link", { name: "Story" });
	expect(link.getAttribute("target")).toBe("_blank");
	const call = fetchMock.mock.calls.find(([url]) =>
		url.endsWith("/sources/preview"),
	);
	expect(JSON.parse(String(call?.[1]?.body))).toEqual({
		type: "medium",
		config: { kind: "profile", identifier: "alice" },
	});
	await user.type(input, "x");
	expect(screen.queryByText("Story")).toBeNull();
});

test("previews a Medium Publication using current kind and identifier", async () => {
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview"))
			return okJson({
				entry: {
					title: "Pub Story",
					link: "https://medium.com/alice/story",
				},
			});
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	await user.selectOptions(screen.getByLabelText("Kind"), "publication");
	await user.type(screen.getByLabelText("Identifier"), "alice");
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByRole("link", { name: "Pub Story" });
	const call = fetchMock.mock.calls.find(([url]) =>
		url.endsWith("/sources/preview"),
	);
	expect(JSON.parse(String(call?.[1]?.body))).toEqual({
		type: "medium",
		config: { kind: "publication", identifier: "alice" },
	});
	await user.selectOptions(screen.getByLabelText("Kind"), "profile");
	expect(screen.queryByText("Pub Story")).toBeNull();
});

test.each([
	"empty",
	"failure",
] as const)("shows Medium preview %s with retry", async (outcome) => {
	let attempts = 0;
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.endsWith("/sources/preview")) {
			attempts++;
			if (attempts === 1)
				return outcome === "empty"
					? okJson({ entry: null })
					: failJson(400, {
							code: "VALIDATION",
							message: "Could not fetch or read the origin",
						});
			return okJson({
				entry: {
					title: "Retried",
					link: "https://medium.com/@alice/retried",
				},
			});
		}
		return emptyList(url, init);
	});
	vi.stubGlobal("fetch", withMediumTypes(fetchMock));
	renderPage();
	const user = userEvent.setup();
	await openMediumAdd(user);
	await user.selectOptions(screen.getByLabelText("Kind"), "profile");
	await user.type(screen.getByLabelText("Identifier"), "alice");
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByText(
		outcome === "empty"
			? "No entries found."
			: "Could not fetch or read the origin",
	);
	await user.click(screen.getByRole("button", { name: "Test fetch" }));
	await screen.findByRole("link", { name: "Retried" });
});

test("edits only the name of a Medium Profile and keeps the target fixed", async () => {
	const mediumSource = {
		id: "md",
		name: "Alice",
		type: "medium",
		config: { kind: "profile", identifier: "alice" },
		active: true,
	};
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("/source-types")) return mediumTypesResponse();
		if (init?.method === "PATCH") {
			return okJson({ ...mediumSource, name: "Renamed" });
		}
		if (isSourcesList(url, init)) {
			return okJson([mediumSource], {
				page: 1,
				pageSize: 10,
				total: 1,
				totalPages: 1,
			});
		}
		return okJson(mediumSource);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "Actions for Alice" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Edit source" }));
	const kind = (await screen.findByLabelText("Kind")) as HTMLSelectElement;
	expect(kind.value).toBe("profile");
	expect(kind.disabled).toBe(true);
	const identifier = screen.getByLabelText("Identifier") as HTMLInputElement;
	expect(identifier.value).toBe("alice");
	expect(identifier.readOnly).toBe(true);
	const name = screen.getByLabelText("Source name");
	await user.clear(name);
	await user.type(name, "Renamed");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "PATCH"),
		).toBe(true),
	);
	const patchCall = fetchMock.mock.calls.find(
		([, init]) => init?.method === "PATCH",
	);
	expect(JSON.parse(String(patchCall?.[1]?.body))).toEqual({
		name: "Renamed",
	});
});

test("edits only the name of a Medium Publication and keeps the target fixed", async () => {
	const mediumSource = {
		id: "md-pub",
		name: "Alice Pub",
		type: "medium",
		config: { kind: "publication", identifier: "alice" },
		active: true,
	};
	const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
		if (url.includes("/source-types")) return mediumTypesResponse();
		if (init?.method === "PATCH") {
			return okJson({ ...mediumSource, name: "Renamed Pub" });
		}
		if (isSourcesList(url, init)) {
			return okJson([mediumSource], {
				page: 1,
				pageSize: 10,
				total: 1,
				totalPages: 1,
			});
		}
		return okJson(mediumSource);
	});
	vi.stubGlobal("fetch", fetchMock);
	renderPage();
	const user = userEvent.setup();
	await user.click(
		await screen.findByRole("button", { name: "Actions for Alice Pub" }),
	);
	await user.click(screen.getByRole("menuitem", { name: "Edit source" }));
	const kind = (await screen.findByLabelText("Kind")) as HTMLSelectElement;
	expect(kind.value).toBe("publication");
	expect(kind.disabled).toBe(true);
	const identifier = screen.getByLabelText("Identifier") as HTMLInputElement;
	expect(identifier.value).toBe("alice");
	expect(identifier.readOnly).toBe(true);
	const name = screen.getByLabelText("Source name");
	await user.clear(name);
	await user.type(name, "Renamed Pub");
	await user.click(screen.getByRole("button", { name: "Save source" }));
	await waitFor(() =>
		expect(
			fetchMock.mock.calls.some(([, init]) => init?.method === "PATCH"),
		).toBe(true),
	);
	expect(
		JSON.parse(
			String(
				fetchMock.mock.calls.find(([, init]) => init?.method === "PATCH")?.[1]
					?.body,
			),
		),
	).toEqual({ name: "Renamed Pub" });
});
