import { useState, useEffect, useCallback } from 'react';
import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
  WindowIsMaximised,
} from '../../../../wailsjs/runtime/runtime';

export function useWindowControls() {
  const [isMaximized, setIsMaximized] = useState(false);

  const checkMaximized = useCallback(() => {
    try {
      WindowIsMaximised()
        .then(setIsMaximized)
        .catch(() => {});
    } catch {
      // Fuera del entorno Wails
    }
  }, []);

  useEffect(() => {
    checkMaximized();
    window.addEventListener('resize', checkMaximized);
    return () => window.removeEventListener('resize', checkMaximized);
  }, [checkMaximized]);

  const minimize = useCallback(() => {
    try {
      WindowMinimise();
    } catch {}
  }, []);

  const toggleMaximize = useCallback(() => {
    try {
      WindowToggleMaximise();
      setTimeout(checkMaximized, 150);
    } catch {}
  }, [checkMaximized]);

  const close = useCallback(() => {
    try {
      Quit();
    } catch {}
  }, []);

  return {
    isMaximized,
    minimize,
    toggleMaximize,
    close,
  };
}
