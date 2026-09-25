// The response statuses offered for endpoints and payloads, with their names,
// and helpers for the response body text box.

export const statuses: { code: number; name: string }[] = [
	{ code: 200, name: 'OK' },
	{ code: 201, name: 'Created' },
	{ code: 400, name: 'Bad Request' },
	{ code: 401, name: 'Unauthorized' },
	{ code: 403, name: 'Forbidden' },
	{ code: 404, name: 'Not Found' },
	{ code: 409, name: 'Conflict' },
	{ code: 418, name: "I'm a teapot" },
	{ code: 422, name: 'Unprocessable Entity' },
	{ code: 500, name: 'Internal Server Error' },
	{ code: 502, name: 'Bad Gateway' },
	{ code: 503, name: 'Service Unavailable' },
	{ code: 504, name: 'Gateway Timeout' }
];

/** A sample response body for a status, formatted for the body text box. */
export function sampleBody(code: number): string {
	const name = statuses.find((status) => status.code === code)?.name ?? '';
	const sample =
		code < 400
			? { message: name }
			: { statusCode: code, error: name, message: 'An error occurred' };
	return JSON.stringify(sample, null, 2);
}

/** A response body as text for the body text box: pretty-printed, or empty if there's none. */
export function bodyText(body: unknown): string {
	return body === undefined ? '' : JSON.stringify(body, null, 2);
}

/**
 * Reads the body text box. Empty means no body (undefined); anything else
 * must be JSON, or this throws an Error with a message for the user.
 */
export function parseBodyText(text: string): unknown {
	if (text.trim() === '') {
		return undefined;
	}
	try {
		return JSON.parse(text);
	} catch {
		throw new Error('The response body is not valid JSON');
	}
}
