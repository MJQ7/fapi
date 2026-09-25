// Every call to fapi's admin API lives in this file, so components never call
// fetch themselves. The types mirror the JSON the Go code sends
// (internal/api and internal/core).

export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

export const methods: Method[] = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

export type Mock = {
	id: string;
	port: number;
	method: Method;
	path: string;
	status: number;
	body?: unknown; // any JSON value; missing when the endpoint has no body
	disabled?: boolean; // true when turned off; missing when on
};

export type NewMock = Omit<Mock, 'id' | 'disabled'>;

export type MocksData = {
	enabled: boolean;
	mocks: Mock[];
	upstreams: Record<string, number>; // fapi port -> real API port
	/** fapi port -> real API host, for proxies not using passThrough.host. */
	upstreamHosts?: Record<string, string>;
	/** The fapi ports whose proxy is turned off. */
	disabledUpstreams?: Record<string, boolean>;
};

/** A saved response, offered when adding an endpoint. */
export type Payload = {
	id: string;
	name: string;
	status: number;
	body?: unknown; // any JSON value; missing when the payload has no body
};

export type NewPayload = Omit<Payload, 'id'>;

export type LoggedRequest = {
	id: string;
	time: string;
	port: number;
	method: string;
	path: string;
	search: string;
	contentType: string;
	body: string;
	bodyTruncated: boolean;
	status: number;
	outcome: 'mocked' | 'proxied' | 'unmatched' | 'preflight';
	mockId?: string;
	upstream?: number;
	fromPort: number;
};

export type Settings = {
	adminPort: number;
	listenAddress: string;
	mockPorts: PortRange;
	features: {
		webUi: boolean;
		passThrough: boolean;
		cors: boolean;
		requestLog: boolean;
		requestLogPersistence: boolean;
		liveUpdates: boolean;
	};
	requestLog: { maxEntries: number; maxBodyBytes: number };
	passThrough: { host: string };
};

/** The range of ports endpoints and proxies may use, and the one suggested for new ones. */
export type PortRange = { min: number; max: number; default: number };

/** The settings on the Settings screen. */
export type Ports = { adminPort: number; mockPorts: PortRange };

/**
 * The port settings fapi is using, and those saved for its next start.
 * They differ (restartNeeded) until fapi restarts.
 */
export type PortSettings = {
	file: string;
	active: Ports;
	saved: Ports;
	restartNeeded: boolean;
};

/** Whether GitHub has a newer release of fapi (internal/updates). */
export type UpdateCheck = {
	current: string; // the running version, such as "1.2.3" or "dev"
	latest: string; // the newest release, or '' if the check failed or there are none
	latestUrl: string;
	publishedAt: string;
	updateAvailable: boolean;
	/** The running version isn't a release number, so it can't be compared. */
	developmentBuild: boolean;
	checkedAt: string;
	error?: string; // why the check failed
};

/** An error from the admin API, with its message worded for the user. */
export class ApiError extends Error {}

export function getConfig(): Promise<Settings> {
	return send<Settings>('GET', '/api/config');
}

export function getPortSettings(): Promise<PortSettings> {
	return send<PortSettings>('GET', '/api/settings/ports');
}

/** Saves port settings to fapi's settings file. They take effect when fapi restarts. */
export function savePortSettings(ports: Ports): Promise<PortSettings> {
	return send<PortSettings>('PUT', '/api/settings/ports', ports);
}

/**
 * Asks fapi whether there's a newer release on GitHub. fapi remembers the
 * answer for an hour; force asks GitHub again.
 */
export function checkForUpdates(force = false): Promise<UpdateCheck> {
	return send<UpdateCheck>('GET', force ? '/api/updates?force=true' : '/api/updates');
}

export function getMocks(): Promise<MocksData> {
	return send<MocksData>('GET', '/api/mocks');
}

export function addMock(mock: NewMock): Promise<Mock> {
	return send<Mock>('POST', '/api/mocks', mock);
}

export async function deleteMock(id: string): Promise<void> {
	await send('DELETE', `/api/mocks/${encodeURIComponent(id)}`);
}

/** Turns one endpoint on or off. */
export async function setMockEnabled(id: string, enabled: boolean): Promise<void> {
	await send('PUT', `/api/mocks/${encodeURIComponent(id)}/enabled`, { enabled });
}

/** Turns the proxy on a fapi port on or off. */
export async function setUpstreamEnabled(port: number, enabled: boolean): Promise<void> {
	await send('PUT', `/api/upstreams/${port}/enabled`, { enabled });
}

export async function setEnabled(enabled: boolean): Promise<void> {
	await send('PUT', '/api/enabled', { enabled });
}

/**
 * Sets the real API host and port for a fapi port. A null port removes the
 * proxy. An empty host means passThrough.host in fapi's settings; a host
 * starting with https:// is reached over HTTPS.
 */
export async function setUpstream(
	port: number,
	upstreamPort: number | null,
	host = ''
): Promise<void> {
	await send('PUT', `/api/upstreams/${port}`, { upstreamPort, host });
}

/** A saved proxy: a fapi port, where its requests are forwarded to, and whether it's on. */
export type Proxy = { port: number; realAPI: string; enabled: boolean };

/**
 * The saved proxies, in port order. defaultHost is passThrough.host in fapi's
 * settings: the host of proxies without one of their own.
 */
export function listProxies(data: MocksData, defaultHost: string): Proxy[] {
	const hosts = data.upstreamHosts ?? {};
	const disabled = data.disabledUpstreams ?? {};
	return Object.entries(data.upstreams)
		.map(([port, upstreamPort]) => ({
			port: Number(port),
			realAPI: upstreamURL(hosts[port] || defaultHost, upstreamPort),
			enabled: !disabled[port]
		}))
		.sort((a, b) => a.port - b.port);
}

/**
 * Where a proxy forwards to, such as "http://localhost:3000" or
 * "https://api.example.com". Mirrors core.UpstreamURL in the Go code.
 */
export function upstreamURL(host: string, port: number): string {
	const https = host.startsWith('https://');
	const scheme = https ? 'https' : 'http';
	let name = https ? host.slice('https://'.length) : host;
	if (name.includes(':')) {
		name = `[${name}]`; // an IPv6 address
	}
	return port === (https ? 443 : 80) ? `${scheme}://${name}` : `${scheme}://${name}:${port}`;
}

export function getPayloads(): Promise<Payload[]> {
	return send<Payload[]>('GET', '/api/payloads');
}

export function addPayload(payload: NewPayload): Promise<Payload> {
	return send<Payload>('POST', '/api/payloads', payload);
}

export function updatePayload(id: string, payload: NewPayload): Promise<Payload> {
	return send<Payload>('PUT', `/api/payloads/${encodeURIComponent(id)}`, payload);
}

export async function deletePayload(id: string): Promise<void> {
	await send('DELETE', `/api/payloads/${encodeURIComponent(id)}`);
}

export function getRequests(): Promise<LoggedRequest[]> {
	return send<LoggedRequest[]>('GET', '/api/requests');
}

export async function clearRequests(): Promise<void> {
	await send('DELETE', '/api/requests');
}

type RequestWatcher = {
	onRequest: (request: LoggedRequest) => void;
	onCleared: () => void;
	/** Called on every (re)connection, when events may have been missed. */
	onConnected: () => void;
};

/**
 * Listens for new requests with server-sent events, and returns a function
 * that stops listening. The browser reconnects by itself if the connection
 * drops.
 */
export function watchRequests(watcher: RequestWatcher): () => void {
	const events = new EventSource('/api/requests/stream');
	events.addEventListener('open', () => watcher.onConnected());
	events.addEventListener('request', (event) => {
		watcher.onRequest(JSON.parse((event as MessageEvent).data));
	});
	events.addEventListener('cleared', () => watcher.onCleared());
	return () => events.close();
}

/** Sends one request to the admin API and returns the JSON response, if any. */
async function send<T = void>(method: string, path: string, body?: unknown): Promise<T> {
	let response: Response;
	try {
		response = await fetch(path, {
			method,
			headers: body === undefined ? undefined : { 'content-type': 'application/json' },
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	} catch {
		throw new ApiError('Could not reach fapi. Is it still running?');
	}

	if (!response.ok) {
		throw new ApiError(await errorMessage(response));
	}
	if (response.status === 204) {
		return undefined as T;
	}
	return (await response.json()) as T;
}

/** The API sends errors as {"message": "..."}; fall back to the status. */
async function errorMessage(response: Response): Promise<string> {
	try {
		const body = await response.json();
		if (typeof body.message === 'string') {
			return body.message;
		}
	} catch {
		// Not JSON: use the status below.
	}
	return `fapi answered ${response.status} ${response.statusText}`;
}
