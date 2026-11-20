import { useEffect, useState } from 'react';
import { GetTodayAIUsage } from '../../../../wailsjs/go/app/App';
import { EventsOn } from '../../../../wailsjs/runtime/runtime';
import type { domain } from '../../../../wailsjs/go/models';
import type { ChatStreamEvent } from '../../../types';
import { formatCost } from '../../../formatCost';

export function DailyUsageCards({ providerId }: { providerId: string }) {
  const [usage, setUsage] = useState<domain.DailyProviderUsage>();
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    let sequence = 0;
    let timer: number;
    const refresh = () => {
      const request = ++sequence;
      GetTodayAIUsage(providerId).then((data) => {
        if (!active || request !== sequence) return;
        setUsage(data);
        setError(false);
      }).catch(() => {
        if (!active || request !== sequence) return;
        setUsage(undefined);
        setError(true);
      });
    };
    const scheduleDay = () => {
      const midnight = new Date();
      midnight.setHours(24, 0, 0, 0);
      timer = window.setTimeout(() => {
        refresh();
        scheduleDay();
      }, midnight.getTime() - Date.now());
    };
    const unsubscribe = EventsOn('chat:stream', (event: ChatStreamEvent) => {
      if (event?.providerId === providerId && (event.type === 'done' || event.type === 'error')) {
        refresh();
      }
    });
    refresh();
    scheduleDay();
    return () => {
      active = false;
      window.clearTimeout(timer);
      unsubscribe();
    };
  }, [providerId]);

  const cost = usage?.cost;
  const amount = cost?.requests === 0 ? 0 : cost?.nanosUsd;
  const cards = [
    { label: 'Tokens de entrada', value: usage?.tokensPrompt.toLocaleString() },
    { label: 'Tokens de salida', value: usage?.tokensCompletion.toLocaleString() },
    { label: 'Costo estimado', value: amount == null ? undefined : formatCost(amount) },
  ];

  return (
    <section aria-label="Consumo de hoy">
      <h3 className="mb-3 text-xs font-semibold tracking-wider text-content-headline uppercase">
        Consumo de hoy
      </h3>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3" aria-busy={!usage && !error}>
        {cards.map(({ label, value }) => (
          <div key={label} className="flex flex-col gap-2 p-4 rounded-xl border border-border-subtle/60 bg-surface-lowest">
            <span className="text-xs text-content-muted font-medium">{label}</span>
            <span className="text-lg font-semibold font-mono text-content-headline">{value ?? '—'}</span>
          </div>
        ))}
      </div>
      {error && <p className="mt-2 text-xs text-content-muted" role="status">No se pudo cargar el consumo de hoy.</p>}
      {cost?.partial && (
        <p className="mt-2 text-xs text-content-muted">
          {amount == null ? 'Costo no disponible para las llamadas registradas.' : 'Costo parcial. Algunas llamadas no se pudieron valorar.'}
        </p>
      )}
      {providerId === 'google' && (
        <p className="mt-2 text-xs text-content-muted">La estimación de Gemini usa la tarifa del plan de pago.</p>
      )}
    </section>
  );
}
