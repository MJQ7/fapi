import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

// The admin API of the fapi you're developing against (see README.md).
const fapiAdminUrl = 'http://127.0.0.1:3100';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// Build plain files that fapi embeds in its executable and serves
			// itself, so no Node server runs with fapi. Every page is rendered in
			// the browser, so paths that aren't files get index.html.
			// build/README.md stays, which lets Go build before the UI is built.
			adapter: adapter({
				pages: 'build/ui',
				assets: 'build/ui',
				fallback: 'index.html'
			})
		})
	],
	server: {
		// In development, send /api to a running fapi, so the browser sees one
		// origin. The admin API refuses requests from other origins, and the
		// dev server is one, so the proxy tells fapi the request comes from
		// fapi's own origin (Origin header) at fapi's own address (Host header,
		// set by changeOrigin).
		proxy: {
			'/api': {
				target: fapiAdminUrl,
				changeOrigin: true,
				headers: { origin: fapiAdminUrl }
			}
		}
	}
});
