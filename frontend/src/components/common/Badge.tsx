import { forwardRef, HTMLAttributes } from 'react';
import { cn } from '@/utils/cn';

export type BadgeStatus =
  | 'PENDING'
  | 'RUNNING'
  | 'SUCCEEDED'
  | 'FAILED'
  | 'UNKNOWN'
  | 'default';

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  /** Predefined status style */
  status?: BadgeStatus;
  /** Custom variant style */
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
  /** Size of the badge */
  size?: 'sm' | 'md' | 'lg';
  /** Whether to use a dot indicator */
  withDot?: boolean;
  /** Content to display */
  children: React.ReactNode;
}

const statusToVariant: Record<BadgeStatus, BadgeProps['variant']> = {
  PENDING: 'warning',
  RUNNING: 'info',
  SUCCEEDED: 'success',
  FAILED: 'error',
  UNKNOWN: 'default',
  default: 'default',
};

const variantStyles = {
  default:
    'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200',
  primary:
    'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  success:
    'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  warning:
    'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  error: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  info: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-300',
};

const dotColors = {
  default: 'bg-gray-400',
  primary: 'bg-blue-500',
  success: 'bg-green-500',
  warning: 'bg-yellow-500',
  error: 'bg-red-500',
  info: 'bg-indigo-500',
};

const sizes = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-0.5 text-sm',
  lg: 'px-3 py-1 text-sm',
};

/**
 * Badge component for displaying status and labels.
 * Supports predefined status values and custom variants.
 */
const Badge = forwardRef<HTMLSpanElement, BadgeProps>(
  (
    {
      className,
      status,
      variant,
      size = 'md',
      withDot = false,
      children,
      ...props
    },
    ref
  ) => {
    const resolvedVariant = variant || statusToVariant[status || 'default'];

    return (
      <span
        ref={ref}
        className={cn(
          'inline-flex items-center font-medium rounded-full',
          sizes[size],
          variantStyles[resolvedVariant || 'default'],
          className
        )}
        {...props}
      >
        {withDot && (
          <span
            className={cn(
              'mr-1.5 h-2 w-2 rounded-full',
              dotColors[resolvedVariant || 'default']
            )}
            aria-hidden="true"
          />
        )}
        {children}
      </span>
    );
  }
);

Badge.displayName = 'Badge';

export { Badge };
