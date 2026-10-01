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
	proxies: SavedProxy[]; // in the order added
};

/**
 * A proxy as fapi saves it (core.Proxy). A port can have several, but only
 * one is on at a time.
 */
export type SavedProxy = {
	id: string;
	port: number; // the fapi port
	upstreamPort: number; // the real API port
	host?: string; // the real API host; missing for passThrough.host
	disabled?: boolean; // true when off; missing when on
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
	proxyId?: string; // the proxy that forwarded it, when proxied
	fromPort: number;
	/**
	 * What the sender got back: the real API's answer when proxied, otherwise
	 * fapi's own. Missing in logs saved by older versions.
	 */
	response?: LoggedResponse;
};

export type LoggedResponse = {
	contentType: string;
	body: string;
	bodyTruncated: boolean;
	/** Why the real API couldn't be reached, when proxied; fapi then answered 502 and there's no body. */
	error?: string;
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
		installUpdates: boolean;
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
	install: Installation;
	/** The file an update would download, such as "fapi_1.2.0_linux_amd64.deb", or ''. */
	asset: string;
};

/** How the running fapi was installed, and whether it can update itself. */
export type Installation = {
	type: 'deb' | 'rpm' | 'windows' | 'docker' | 'manual';
	package: string; // the edition: fapi, fapi-web or fapi-cli
	os: string; // as Go names it, such as "linux" or "windows"
	arch: string; // such as "amd64"
	wsl: boolean; // Linux in Windows Subsystem for Linux
	/** fapi can download, install and restart by itself. */
	canInstall: boolean;
	/** fapi can at least download an update, for you to install. */
	canDownload: boolean;
	reason?: string; // why it can't install updates itself
};

/** An update being downloaded or installed, or how the last one went. */
export type UpdateJob = {
	state: 'idle' | 'downloading' | 'installing' | 'restarting' | 'downloaded' | 'failed';
	version?: string;
	downloaded: number; // bytes so far
	total: number; // bytes in all, or 0 when unknown
	file?: string; // where it was downloaded, on the machine fapi runs on
	command?: string; // installs the downloaded file, when fapi can't
	error?: string;
	/** How the last install went, when none is running. */
	last?: { version: string; ok: boolean; error?: string; time: string };
};

/** The running fapi, from GET /api/status. */
export type Status = {
	pid: number;
	version: string;
	adminPort: number;
	mockPorts: number[];
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

/**
 * Downloads version (the newest release) on the machine fapi runs on and,
 * where it can, installs it and restarts fapi. It returns straight away;
 * getUpdateJob follows the progress.
 */
export function installUpdate(version: string): Promise<UpdateJob> {
	return send<UpdateJob>('POST', '/api/updates/install', { version });
}

export function getUpdateJob(): Promise<UpdateJob> {
	return send<UpdateJob>('GET', '/api/updates/install');
}

export function getStatus(): Promise<Status> {
	return send<Status>('GET', '/api/status');
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

/** Turns a proxy on or off. Turning one on turns the others on its port off. */
export async function setProxyEnabled(id: string, enabled: boolean): Promise<void> {
	await send('PUT', `/api/proxies/${encodeURIComponent(id)}/enabled`, { enabled });
}

export async function setEnabled(enabled: boolean): Promise<void> {
	await send('PUT', '/api/enabled', { enabled });
}

/**
 * Adds a proxy from a fapi port to the real API port on host, and turns it
 * on, turning off any other proxy on its port. If the port already has a
 * proxy to the same place, that one is turned on instead. An empty host
 * means passThrough.host in fapi's settings; a host starting with https:// is
 * reached over HTTPS.
 */
export function addProxy(port: number, upstreamPort: number, host = ''): Promise<SavedProxy> {
	return send<SavedProxy>('POST', '/api/proxies', { port, upstreamPort, host });
}

export async function deleteProxy(id: string): Promise<void> {
	await send('DELETE', `/api/proxies/${encodeURIComponent(id)}`);
}

/** A saved proxy: a fapi port, where its requests are forwarded to, and whether it's on. */
export type Proxy = {
	id: string;
	port: number;
	upstreamPort: number;
	realAPI: string;
	enabled: boolean;
};

/**
 * The saved proxies, in port order, and in the order added within a port.
 * defaultHost is passThrough.host in fapi's settings: the host of proxies
 * without one of their own.
 */
export function listProxies(data: MocksData, defaultHost: string): Proxy[] {
	return data.proxies
		.map((proxy) => ({
			id: proxy.id,
			port: proxy.port,
			upstreamPort: proxy.upstreamPort,
			realAPI: upstreamURL(proxy.host || defaultHost, proxy.upstreamPort),
			enabled: !proxy.disabled
		}))
		.sort((a, b) => a.port - b.port);
}

/**
 * The proxy to show for a port, which has several: the one that's on, or
 * else the last one added. Undefined if the port has none.
 */
export function portProxy(proxies: Proxy[], port: number): Proxy | undefined {
	const onPort = proxies.filter((proxy) => proxy.port === port);
	return onPort.find((proxy) => proxy.enabled) ?? onPort.at(-1);
}

/** Each port's proxy to show (see portProxy), in port order. */
export function portProxies(proxies: Proxy[]): Proxy[] {
	const ports = [...new Set(proxies.map((proxy) => proxy.port))];
	return ports.map((port) => portProxy(proxies, port)!);
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

/** How many requests a port received in one step of a traffic report. */
export type TrafficCounts = {
	received: number; // always proxied + sent
	proxied: number; // forwarded to the real API
	sent: number; // answered by fapi itself: an endpoint or a CORS preflight
};

/** The periods a traffic report can cover, in minutes. */
export type TrafficMinutes = 5 | 15 | 30 | 60;

/** The requests each port received over a period, in 60 equal steps, oldest first. */
export type TrafficReport = {
	start: string; // when the first step began
	stepSeconds: number;
	ports: Record<string, TrafficCounts[]>; // only the ports that received requests
};

/** Counts the requests each port received over the last few minutes, for the dashboard. */
export function getTraffic(minutes: TrafficMinutes): Promise<TrafficReport> {
	return send<TrafficReport>('GET', `/api/traffic?minutes=${minutes}`);
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
