import React, { useState } from 'react';
import { IconChevronDown, IconChevronRight } from '../../../components/Icons';

interface ThoughtChainProps {
  thoughts: string;
}

export const ThoughtChain: React.FC<ThoughtChainProps> = ({ thoughts }) => {
  const [isOpen, setIsOpen] = useState(true);

  if (!thoughts) return null;

  return (
    <div className="w-full mb-2.5 rounded-lg border border-border-petrol bg-surface-neutral overflow-hidden">
      <button
        type="button"
        className="w-full flex items-center gap-2 px-3 py-2 text-xs text-content-muted hover:text-content-body hover:bg-surface-hover transition-colors cursor-pointer text-left"
        onClick={() => setIsOpen((prev) => !prev)}
      >
        {isOpen ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
        <span className="text-[11px] font-mono text-accent-cyan tracking-wider font-semibold">
          CADENA DE RAZONAMIENTO
        </span>
      </button>
      {isOpen && (
        <div className="p-3 text-xs font-mono text-content-dim bg-surface-lowest border-t border-border-petrol whitespace-pre-wrap leading-relaxed">
          {thoughts}
        </div>
      )}
    </div>
  );
};
