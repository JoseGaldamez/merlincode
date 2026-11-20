export type FilePreview =
  | { kind: 'image'; src: string }
  | { kind: 'pdf'; data: Uint8Array<ArrayBuffer> }
  | { kind: 'markdown'; text: string }
  | { kind: 'text'; text: string }
  | { kind: 'unsupported' };

const imageTypes: Record<string, string> = {
  avif: 'image/avif',
  bmp: 'image/bmp',
  gif: 'image/gif',
  ico: 'image/x-icon',
  jpeg: 'image/jpeg',
  jpg: 'image/jpeg',
  png: 'image/png',
  svg: 'image/svg+xml',
  webp: 'image/webp',
};

export function fileNameFromPath(path: string) {
  return path.split(/[\\/]/).at(-1) || path;
}

export function decodeFilePreview(path: string, encoded: string): FilePreview {
  const name = fileNameFromPath(path).toLowerCase();
  const ext = name.includes('.') ? name.slice(name.lastIndexOf('.') + 1) : '';
  if (Object.hasOwn(imageTypes, ext)) {
    return { kind: 'image', src: 'data:' + imageTypes[ext] + ';base64,' + encoded };
  }

  const binary = atob(encoded);
  const data = Uint8Array.from(binary, (byte) => byte.charCodeAt(0));
  if (ext === 'pdf' || binary.slice(0, 1024).includes('%PDF-')) return { kind: 'pdf', data };

  const encoding = data[0] === 0xff && data[1] === 0xfe ? 'utf-16le'
    : data[0] === 0xfe && data[1] === 0xff ? 'utf-16be' : 'utf-8';
  try {
    const text = new TextDecoder(encoding, { fatal: true }).decode(data);
    if (!text.includes('\0')) {
      return { kind: ext === 'md' || ext === 'markdown' ? 'markdown' : 'text', text };
    }
  } catch {
    // Unsupported text encodings and binary formats cannot be displayed as text.
  }
  return { kind: 'unsupported' };
}
