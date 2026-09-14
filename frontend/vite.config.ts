import { randomBytes } from 'node:crypto';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ command }) => {
  // Only the loopback development server needs React Refresh's inline preamble.
  const nonce = command === 'serve' ? randomBytes(24).toString('base64') : undefined;
  return {
    plugins: [react(), {
      name: 'merlin-dev-csp',
      apply: 'serve',
      transformIndexHtml: {
        order: 'post',
        handler: (html) => html
          .replace("script-src 'self';", `script-src 'self' 'nonce-${nonce}';`)
          .replace("connect-src 'self' ipc: wails:;", "connect-src 'self' ipc: wails: ws://127.0.0.1:5173 ws://localhost:5173;"),
      },
    }],
    html: nonce ? { cspNonce: nonce } : undefined,
    server: { host: '127.0.0.1', port: 5173, strictPort: true },
  };
});
