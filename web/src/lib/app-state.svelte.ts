// The data every screen shows (settings, endpoints, payloads and the request
// log), loaded from the admin API once and shared, so switching screens
// doesn't reload it or drop the live request stream. The layout calls load
// and watchRequests; screens read the fields and call the other methods.

import * as api from '$lib/api';

/** How often to check for new requests when live updates are turned off. */
const pollMilliseconds = 2000;

class AppState {
	settings = $state<api.Settings | null>(null);
	data = $state<api.MocksData>({ enabled: true, mocks: [], upstreams: {} });
	payloads = $state<api.Payload[]>([]);
	requests = $state<api.LoggedRequest[]>([]);
	error = $state('');

	features = $derived(this.settings?.features);

	/** The saved proxies in port order, or null when proxies are turned off. */
	proxies = $derived(
		this.features?.passThrough
			? api.listProxies(this.data, this.settings?.passThrough.host ?? 'localhost')
			: null
	);

	async load() {
		try {
			this.settings = await api.getConfig();
			await Promise.all([this.loadMocks(), this.loadPayloads(), this.loadRequests()]);
		} catch (caught) {
			this.error = (caught as Error).message;
		}
	}

	loadMocks = async () => {
		this.data = await api.getMocks();
	};

	loadPayloads = async () => {
		this.payloads = await api.getPayloads();
	};

	loadRequests = async () => {
		if (!this.features?.requestLog) {
			return;
		}
		this.requests = await api.getRequests();
	};

	/**
	 * Keeps the request log up to date: with server-sent events when live
	 * updates are on, otherwise by checking every 2 seconds. Call it in an
	 * $effect, which reruns it when the settings arrive and calls the
	 * function it returns to stop the updates.
	 */
	watchRequests(): (() => void) | undefined {
		const features = this.features;
		if (!features?.requestLog) {
			return;
		}

		if (features.liveUpdates) {
			return api.watchRequests({
				onRequest: (request) => {
					const maxEntries = this.settings?.requestLog.maxEntries ?? 200;
					this.requests = [request, ...this.requests].slice(0, maxEntries);
				},
				onCleared: () => (this.requests = []),
				onConnected: () => this.showErrors(this.loadRequests())
			});
		}

		const timer = setInterval(() => this.showErrors(this.loadRequests()), pollMilliseconds);
		return () => clearInterval(timer);
	}

	/** Runs an API call and shows its error, if it fails. */
	async showErrors(call: Promise<void>) {
		try {
			await call;
			this.error = '';
		} catch (caught) {
			this.error = (caught as Error).message;
		}
	}

	setEnabled = (enabled: boolean) => {
		this.showErrors(api.setEnabled(enabled).then(this.loadMocks));
	};

	setMockEnabled = (id: string, enabled: boolean) => {
		this.showErrors(api.setMockEnabled(id, enabled).then(this.loadMocks));
	};

	setUpstreamEnabled = (port: number, enabled: boolean) => {
		this.showErrors(api.setUpstreamEnabled(port, enabled).then(this.loadMocks));
	};

	deleteMock = (id: string) => {
		this.showErrors(api.deleteMock(id).then(this.loadMocks));
	};

	clearRequests = () => {
		this.showErrors(api.clearRequests().then(this.loadRequests));
	};
}

export const app = new AppState();
