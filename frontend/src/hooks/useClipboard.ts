import { useState, useCallback } from 'react';

export function useClipboard(timeout = 2000) {
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const copy = useCallback(
    (id: string, text: string) => {
      if (!text) return;
      navigator.clipboard.writeText(text);
      setCopiedId(id);
      const timer = setTimeout(() => {
        setCopiedId((current) => (current === id ? null : current));
      }, timeout);
      return () => clearTimeout(timer);
    },
    [timeout]
  );

  const isCopied = useCallback(
    (id: string) => copiedId === id,
    [copiedId]
  );

  return { copiedId, copy, isCopied };
}
