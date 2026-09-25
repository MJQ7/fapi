# fapi web UI

The web UI of fapi: a SvelteKit app with Svelte 5, [shadcn-svelte](https://www.shadcn-svelte.com) components and Tailwind CSS, in TypeScript.

It's built into plain files that fapi embeds in its executable and serves on the admin port. No Node.js server runs with fapi.

## Layout

```
src/routes/+layout.svelte    the sidebar around every screen; starts loading the data
src/routes/**/+page.svelte   one screen each: Endpoints (/), Override an endpoint,
                             Payloads, Proxies and Settings
src/lib/app-state.svelte.ts  the data every screen shares, loaded once from the admin API
src/lib/api.ts               every call to fapi's admin API (components never call fetch)
src/lib/statuses.ts          the response statuses offered, their sample bodies, and
                             reading the body text box
src/lib/format.ts            helpers for showing logged requests
src/lib/components/          fapi's own components, one job each
src/lib/components/ui/       shadcn-svelte components, as generated
src/app.html                 the page shell; follows the system's light or dark theme
src/theme.css                colours, the hero colour and layout sizes; restyle the app here
vite.config.ts               the static build, and the /api proxy for development
embed.go                     the Go file that builds build/ui into the fapi executable
```

To change the look, edit `src/theme.css`: `--hero-hue` recolours the brand colour (buttons, switches, links, the current screen), and the layout variables set the content width and spacing. Components use these names rather than fixed colours or sizes.

The shadcn-svelte components in `src/lib/components/ui/` are kept as generated, so they can be updated later. Add more with:

```bash
npx shadcn-svelte@latest add <component>
```

## Developing

You need Node.js 22 or later, and a running fapi for the UI to talk to.

1. Start fapi in one terminal, from the repository root:

   ```bash
   go run ./cmd/fapi serve
   ```

2. Start the dev server in another, from this folder:

   ```bash
   npm install
   npm run dev
   ```

3. Open the address it prints (http://localhost:5173). Changes to the code show up straight away.

The dev server forwards `/api` to fapi's admin API at `http://127.0.0.1:3100`, so the browser sees one origin. The admin API refuses requests from other origins, so the forwarded requests are sent with fapi's own origin (see `vite.config.ts`). If your fapi uses another admin port, change `fapiAdminUrl` in `vite.config.ts`.

Check types and Svelte code with:

```bash
npm run check
```

## Building

```bash
npm run build
```

This writes the UI to `build/ui/`. Then build fapi from the repository root, which embeds it:

```bash
go build ./cmd/fapi
```

`build/README.md` is committed so the `build` folder always exists: Go can then build fapi before the UI has been built, and that fapi shows a page explaining how to build the UI.
