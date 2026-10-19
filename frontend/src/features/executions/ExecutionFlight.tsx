import React, { useEffect, useRef } from 'react';
import { IconPlay } from '../../components/Icons';
import type { ExecutionFocus } from './types';

export function ExecutionFlight({ focus }: { focus?: ExecutionFocus }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const element = ref.current;
    if (!element || !focus?.source || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    let animation: Animation | undefined;
    const frame = requestAnimationFrame(() => {
      const target = document.querySelector('[data-execution-destination]')?.getBoundingClientRect();
      if (!target) return;
      const source = focus.source!;
      const x = source.x+source.width/2;
      const y = source.y+source.height/2;
      animation = element.animate([
        { transform: `translate(${x}px, ${y}px) scale(.85)`, opacity: 0 },
        { transform: `translate(${x+24}px, ${y-20}px) scale(1)`, opacity: 1, offset: .18 },
        { transform: `translate(${target.x+16}px, ${target.y+60}px) scale(.65)`, opacity: 0 },
      ], { duration: 620, easing: 'cubic-bezier(.16,1,.3,1)' });
    });
    return () => { cancelAnimationFrame(frame); animation?.cancel(); };
  }, [focus]);
  return <div ref={ref} className="execution-flight" aria-hidden="true"><IconPlay size={13} /><code>{focus?.command}</code></div>;
}
