// Small helpers for showing logged requests.

/** The time of day a request arrived, in the browser's locale. */
export function formatTime(time: string): string {
	return new Date(time).toLocaleTimeString();
}

/** A request body for display: pretty-printed if it's JSON, as it arrived if not. */
export function formatBody(body: string): string {
	try {
		return JSON.stringify(JSON.parse(body), null, 2);
	} catch {
		return body;
	}
}

/** "1 request", "2 requests". */
export function countRequests(count: number): string {
	return count === 1 ? '1 request' : `${count} requests`;
}
