import { clsx } from 'clsx';
import type { ReactNode } from 'react';

export function Badge({ children, tone = 'neutral' }: { children: ReactNode; tone?: 'neutral' | 'ok' | 'warn' | 'danger' }) {
  return (
    <span
      className={clsx(
        'inline-flex min-h-6 items-center rounded-md border px-2 py-0.5 text-xs font-medium',
        tone === 'neutral' && 'border-border bg-muted text-muted-foreground',
        tone === 'ok' && 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300',
        tone === 'warn' && 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-300',
        tone === 'danger' && 'border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300',
      )}
    >
      {children}
    </span>
  );
}
