import { useRef, useEffect } from 'react';

interface AutoResizeOptions {
  minHeight?: number;
  maxHeight?: number;
}

export function useAutoResizeTextarea(
  value: string,
  options: AutoResizeOptions = {}
) {
  const { minHeight = 40, maxHeight = 112 } = options;
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;

    el.style.height = 'auto';
    const scrollH = el.scrollHeight;

    if (!value) {
      el.style.height = `${minHeight}px`;
      el.style.overflowY = 'hidden';
    } else if (scrollH <= maxHeight) {
      el.style.height = `${Math.max(minHeight, scrollH)}px`;
      el.style.overflowY = 'hidden';
    } else {
      el.style.height = `${maxHeight}px`;
      el.style.overflowY = 'auto';
    }
  }, [value, minHeight, maxHeight]);

  return textareaRef;
}
