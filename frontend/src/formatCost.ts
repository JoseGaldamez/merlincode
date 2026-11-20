const usdFormatter = new Intl.NumberFormat('es', {
  style: 'currency',
  currency: 'USD',
  currencyDisplay: 'code',
  maximumFractionDigits: 6,
});

export function formatCost(nanosUsd: number): string {
  if (nanosUsd > 0 && nanosUsd < 1_000) {
    return `< ${usdFormatter.format(0.000001)}`;
  }
  return usdFormatter.format(nanosUsd / 1_000_000_000);
}
