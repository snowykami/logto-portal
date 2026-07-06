import { clsx } from 'clsx';
import type { ButtonHTMLAttributes, ReactNode } from 'react';

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  icon?: ReactNode;
};

export function Button({ className, variant = 'secondary', icon, children, ...props }: ButtonProps) {
  return (
    <button
      className={clsx(
        'inline-flex h-10 items-center justify-center gap-2 rounded-md px-3 text-sm font-medium transition focus:outline-none focus:ring-2 focus:ring-primary disabled:cursor-not-allowed disabled:opacity-50',
        variant === 'primary' && 'bg-primary text-primary-foreground hover:opacity-92',
        variant === 'secondary' && 'border border-border bg-card text-card-foreground hover:bg-muted',
        variant === 'ghost' && 'text-foreground hover:bg-muted',
        variant === 'danger' && 'bg-danger text-white hover:opacity-90',
        className,
      )}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
}
