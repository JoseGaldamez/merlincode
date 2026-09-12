import { useState, useEffect, useCallback } from 'react';
import {
  SavePanelsState,
  GetPanelsState,
  SaveWindowSize,
} from '../../../../wailsjs/go/main/App';

const STORAGE_KEY_LEFT = 'merlin_left_panel_open';
const STORAGE_KEY_RIGHT = 'merlin_right_panel_open';
const STORAGE_KEY_LEFT_WIDTH = 'merlin_left_panel_width';
const STORAGE_KEY_RIGHT_WIDTH = 'merlin_right_panel_width';

function getInitialPanelState(key: string, defaultVal: boolean): boolean {
  try {
    const saved = localStorage.getItem(key);
    if (saved !== null) {
      return saved === 'true';
    }
  } catch {}
  return defaultVal;
}

function getInitialPanelWidth(key: string, defaultVal: number): number {
  try {
    const saved = localStorage.getItem(key);
    if (saved !== null) {
      const val = parseInt(saved, 10);
      if (!isNaN(val) && val >= 180 && val <= 360) {
        return val;
      }
    }
  } catch {}
  return defaultVal;
}

export function usePanelsState() {
  const [leftOpen, setLeftOpen] = useState<boolean>(() =>
    getInitialPanelState(STORAGE_KEY_LEFT, true)
  );
  const [rightOpen, setRightOpen] = useState<boolean>(() =>
    getInitialPanelState(STORAGE_KEY_RIGHT, false)
  );
  const [leftWidth, setLeftWidth] = useState<number>(() =>
    getInitialPanelWidth(STORAGE_KEY_LEFT_WIDTH, 260)
  );
  const [rightWidth, setRightWidth] = useState<number>(() =>
    getInitialPanelWidth(STORAGE_KEY_RIGHT_WIDTH, 280)
  );

  // Sincronizar estado inicial guardado en backend de Go si no existe en localStorage
  useEffect(() => {
    GetPanelsState()
      .then((state) => {
        if (state) {
          if (localStorage.getItem(STORAGE_KEY_LEFT) === null && typeof state.leftOpen === 'boolean') {
            setLeftOpen(state.leftOpen);
          }
          if (localStorage.getItem(STORAGE_KEY_RIGHT) === null && typeof state.rightOpen === 'boolean') {
            setRightOpen(state.rightOpen);
          }
        }
      })
      .catch(() => {});
  }, []);

  // Guardar automáticamente dimensiones de la ventana al redimensionar
  useEffect(() => {
    let timeoutId: number;
    const handleResize = () => {
      window.clearTimeout(timeoutId);
      timeoutId = window.setTimeout(() => {
        if (typeof window !== 'undefined' && window.outerWidth && window.outerHeight) {
          const isMax =
            window.screen &&
            window.outerWidth >= window.screen.availWidth &&
            window.outerHeight >= window.screen.availHeight;
          SaveWindowSize(window.outerWidth, window.outerHeight, !!isMax).catch(() => {});
        }
      }, 400);
    };

    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
      window.clearTimeout(timeoutId);
    };
  }, []);

  const toggleLeft = useCallback(() => {
    setLeftOpen((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY_LEFT, String(next));
      } catch {}
      SavePanelsState(next, rightOpen).catch(() => {});
      return next;
    });
  }, [rightOpen]);

  const toggleRight = useCallback(() => {
    setRightOpen((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY_RIGHT, String(next));
      } catch {}
      SavePanelsState(leftOpen, next).catch(() => {});
      return next;
    });
  }, [leftOpen]);

  const handleLeftWidthChange = useCallback((w: number) => {
    setLeftWidth(w);
    try {
      localStorage.setItem(STORAGE_KEY_LEFT_WIDTH, String(w));
    } catch {}
  }, []);

  const handleRightWidthChange = useCallback((w: number) => {
    setRightWidth(w);
    try {
      localStorage.setItem(STORAGE_KEY_RIGHT_WIDTH, String(w));
    } catch {}
  }, []);

  return {
    leftOpen,
    rightOpen,
    leftWidth,
    rightWidth,
    toggleLeft,
    toggleRight,
    setLeftWidth: handleLeftWidthChange,
    setRightWidth: handleRightWidthChange,
  };
}
