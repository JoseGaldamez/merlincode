import { useState, useEffect, useCallback } from 'react';

export function useLocalStorage<T>(
  key: string,
  initialValue: T | (() => T)
): [T, (value: T | ((val: T) => T)) => void] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = localStorage.getItem(key);
      if (item !== null) {
        return JSON.parse(item);
      }
    } catch {
      // Si falla o no hay parseo JSON, usar valor inicial
    }
    return typeof initialValue === 'function'
      ? (initialValue as () => T)()
      : initialValue;
  });

  const setValue = useCallback(
    (value: T | ((val: T) => T)) => {
      try {
        setStoredValue((prev) => {
          const valueToStore =
            typeof value === 'function' ? (value as (val: T) => T)(prev) : value;
          try {
            localStorage.setItem(key, JSON.stringify(valueToStore));
          } catch {
            // localStorage lleno o deshabilitado
          }
          return valueToStore;
        });
      } catch {
        // Ignorar errores de serialización
      }
    },
    [key]
  );

  return [storedValue, setValue];
}
