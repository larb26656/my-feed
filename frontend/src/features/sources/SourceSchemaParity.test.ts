import Ajv2020 from "ajv/dist/2020";
import { expect, test } from "vitest";
import cases from "../../../../testdata/source-config-cases.json";
import customSchema from "../../../../testdata/source-custom-schema.json";
import { sourceTypesResponse } from "./sourceTypesTestFixture";

test("shared config cases agree with the published schemas", async () => {
	const response = await sourceTypesResponse();
	const body = await response.json();
	const types = (body.data as { types: { code: string; schema: object }[] })
		.types;
	for (const item of cases) {
		const schema =
			item.type === "custom"
				? customSchema
				: types.find((type) => type.code === item.type)?.schema;
		if (!schema) throw new Error(`Missing schema: ${item.type}`);
		expect(
			new Ajv2020().compile(schema)(item.config),
			JSON.stringify(item),
		).toBe(item.valid);
	}
});
