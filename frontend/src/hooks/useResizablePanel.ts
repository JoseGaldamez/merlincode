import React, { useCallback, useEffect, useRef } from 'react';

export interface UseResizablePanelOptions {
  minWidth?: number;
  maxWidth?: number;
  side: 'left' | 'right';
  onWidthChange: (width: number) => void;
}

export function useResizablePanel({
  minWidth = 180,
  maxWidth = 360,
  side,
  onWidthChange,
}: UseResizablePanelOptions) {
  const isDraggingRef = useRef(false);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      isDraggingRef.current = true;
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';

      const onMouseMove = (moveEvent: MouseEvent) => {
        if (!isDraggingRef.current) return;
        const newWidth =
          side === 'left'
            ? moveEvent.clientX
            : window.innerWidth - moveEvent.clientX;

        const clamped = Math.max(minWidth, Math.min(maxWidth, newWidth));
        onWidthChange(clamped);
      };

      const onMouseUp = () => {
        isDraggingRef.current = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        window.removeEventListener('mousemove', onMouseMove);
        window.removeEventListener('mouseup', onMouseUp);
      };

      window.addEventListener('mousemove', onMouseMove);
      window.addEventListener('mouseup', onMouseUp);
    },
    [minWidth, maxWidth, side, onWidthChange]
  );

  useEffect(() => {
    // Cleanup de seguridad al desmontar
    return () => {
      if (isDraggingRef.current) {
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      }
    };
  }, []);

  return { handleMouseDown };
}
