import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createServer } from 'vite';
import { expect, it } from 'vitest';

it('authorizes only the generated development preamble and leaves production strict', async () => {
  const server = await createServer({ server: { middlewareMode: true, watch: null } });
  try {
    const source = readFileSync(resolve('index.html'), 'utf8');
    const html = await server.transformIndexHtml('/', source);
    const nonce = html.match(/script-src 'self' 'nonce-([^']+)'/)?.[1];
    expect(nonce).toBeTruthy();
    const scripts = [...html.matchAll(/<script([^>]*)>([\s\S]*?)<\/script>/g)];
    const inlineScripts = scripts.filter((match) => !match[1].includes('src='));
    expect(inlineScripts.length).toBeGreaterThan(0);
    for (const script of inlineScripts) expect(script[1]).toContain(`nonce="${nonce}"`);
    expect(html).not.toMatch(/script-src[^;]*unsafe-inline/);
    expect(source).not.toContain('nonce-');
    expect(source).not.toContain('ws://');
  } finally { await server.close(); }
});
