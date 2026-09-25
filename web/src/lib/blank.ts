// Copies a JSON body with its values emptied, keeping its keys and shape, as
// a template to fill in. Used by the dashboard's "Copy with blank values"
// button.

/**
 * A JSON body with blank values, pretty-printed, or null when the body isn't
 * JSON (including a body cut at fapi's size limit). Strings become "",
 * numbers 0 and booleans false; null stays null. An array keeps one blank
 * item, with the keys of all its items.
 */
export function blankValuesFor(body: string): string | null {
	let value: unknown;
	try {
		value = JSON.parse(body);
	} catch {
		return null;
	}
	return JSON.stringify(blank(value), null, 2);
}

function blank(value: unknown): unknown {
	if (value === null) {
		return null;
	}
	if (Array.isArray(value)) {
		return value.length === 0 ? [] : [value.map(blank).reduce(combine)];
	}
	switch (typeof value) {
		case 'object': {
			const blanked: { [key: string]: unknown } = {};
			for (const [key, property] of Object.entries(value as object)) {
				blanked[key] = blank(property);
			}
			return blanked;
		}
		case 'number':
			return 0;
		case 'boolean':
			return false;
		default:
			return '';
	}
}

/**
 * Two blank array items as one: objects get the keys of both. Otherwise the
 * first is kept, unless it's null and the second isn't.
 */
function combine(a: unknown, b: unknown): unknown {
	if (isObject(a) && isObject(b)) {
		const combined = { ...a };
		for (const [key, value] of Object.entries(b)) {
			combined[key] = key in combined ? combine(combined[key], value) : value;
		}
		return combined;
	}
	if (Array.isArray(a) && Array.isArray(b)) {
		const items = [...a, ...b];
		return items.length === 0 ? [] : [items.reduce(combine)];
	}
	return a === null ? b : a;
}

function isObject(value: unknown): value is { [key: string]: unknown } {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}
