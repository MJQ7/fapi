// Describes the structure of a JSON body as a JSON Schema: the type of each
// value and the properties of each object, without the values themselves.
// Used by the dashboard's "Copy schema" button.

type Schema = { [keyword: string]: unknown };

/**
 * The JSON Schema of a JSON body, pretty-printed, or null when the body
 * isn't JSON (including a body cut at fapi's size limit).
 */
export function jsonSchemaFor(body: string): string | null {
	let value: unknown;
	try {
		value = JSON.parse(body);
	} catch {
		return null;
	}
	const schema = { $schema: 'https://json-schema.org/draft/2020-12/schema', ...infer(value) };
	return JSON.stringify(schema, null, 2);
}

function infer(value: unknown): Schema {
	if (value === null) {
		return { type: 'null' };
	}
	if (Array.isArray(value)) {
		// Every item is described by one schema, merged from them all.
		return { type: 'array', items: value.map(infer).reduce(merge, {}) };
	}
	switch (typeof value) {
		case 'object': {
			const properties: { [name: string]: Schema } = {};
			for (const [name, property] of Object.entries(value as object)) {
				properties[name] = infer(property);
			}
			return { type: 'object', properties, required: Object.keys(properties) };
		}
		case 'number':
			return { type: Number.isInteger(value) ? 'integer' : 'number' };
		default:
			return { type: typeof value }; // 'string' or 'boolean'
	}
}

/**
 * One schema that fits values of both a and b, such as two items of an
 * array. Objects keep every property, and require only those both have.
 * Values of different types become anyOf.
 */
function merge(a: Schema, b: Schema): Schema {
	// {} matches anything: the starting point, and the items of an empty array.
	if (Object.keys(a).length === 0) {
		return b;
	}
	if (Object.keys(b).length === 0) {
		return a;
	}
	const variants = [...(a.anyOf ? (a.anyOf as Schema[]) : [a])];
	for (const variant of b.anyOf ? (b.anyOf as Schema[]) : [b]) {
		const same = variants.findIndex((existing) => kind(existing) === kind(variant));
		if (same === -1) {
			variants.push(variant);
		} else {
			variants[same] = mergeSameKind(variants[same], variant);
		}
	}
	return variants.length === 1 ? variants[0] : { anyOf: variants };
}

/** Integers and other numbers merge into one kind. */
function kind(schema: Schema): unknown {
	return schema.type === 'integer' ? 'number' : schema.type;
}

function mergeSameKind(a: Schema, b: Schema): Schema {
	if (a.type === 'object') {
		const aProperties = a.properties as { [name: string]: Schema };
		const bProperties = b.properties as { [name: string]: Schema };
		const properties = { ...aProperties };
		for (const [name, schema] of Object.entries(bProperties)) {
			properties[name] = name in properties ? merge(properties[name], schema) : schema;
		}
		const bRequired = b.required as string[];
		const required = (a.required as string[]).filter((name) => bRequired.includes(name));
		return { type: 'object', properties, required };
	}
	if (a.type === 'array') {
		return { type: 'array', items: merge(a.items as Schema, b.items as Schema) };
	}
	if (a.type !== b.type) {
		return { type: 'number' }; // an integer and a number with a fraction
	}
	return a;
}
