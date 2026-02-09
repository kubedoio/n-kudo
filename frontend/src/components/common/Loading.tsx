import { cn } from '@/utils/cn';

export interface LoadingSpinnerProps {
  /** Size of the spinner */
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  /** Color variant */
  variant?: 'primary' | 'secondary' | 'white' | 'gray';
  /** Additional CSS classes */
  className?: string;
  /** Accessible label for screen readers */
  label?: string;
}

const sizes = {
  xs: 'h-3 w-3 border',
  sm: 'h-4 w-4 border',
  md: 'h-6 w-6 border-2',
  lg: 'h-8 w-8 border-2',
  xl: 'h-12 w-12 border-[3px]',
};

const colors = {
  primary: 'border-blue-600 border-t-transparent',
  secondary: 'border-gray-400 border-t-transparent',
  white: 'border-white border-t-transparent/30',
  gray: 'border-gray-300 border-t-transparent',
};

/**
 * Loading spinner component with multiple sizes and colors.
 * Uses CSS animation for smooth rotation.
 */
export function LoadingSpinner({
  size = 'md',
  variant = 'primary',
  className,
  label = 'Loading...',
}: LoadingSpinnerProps) {
  return (
    <div
      role="status"
      aria-label={label}
      className={cn('inline-block', className)}
    >
      <div
        className={cn(
          'animate-spin rounded-full',
          sizes[size],
          colors[variant]
        )}
        aria-hidden="true"
      />
      <span className="sr-only">{label}</span>
    </div>
  );
}

export interface SkeletonProps {
  /** Width of the skeleton (CSS value) */
  width?: string | number;
  /** Height of the skeleton (CSS value) */
  height?: string | number;
  /** Whether to use a circular shape */
  circle?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Skeleton placeholder component for loading states.
 * Displays an animated pulse placeholder.
 */
export function Skeleton({
  width,
  height,
  circle = false,
  className,
}: SkeletonProps) {
  const style: React.CSSProperties = {
    width: width,
    height: height,
  };

  return (
    <div
      className={cn(
        'animate-pulse bg-gray-200 dark:bg-gray-700',
        circle ? 'rounded-full' : 'rounded-md',
        className
      )}
      style={style}
      aria-hidden="true"
    />
  );
}

export interface SkeletonTextProps {
  /** Number of lines to render */
  lines?: number;
  /** Width of each line (can be array for different widths) */
  widths?: string[] | string;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Skeleton text component for paragraph loading states.
 */
export function SkeletonText({
  lines = 3,
  widths,
  className,
}: SkeletonTextProps) {
  const getWidth = (index: number): string => {
    if (Array.isArray(widths)) {
      return widths[index] || '100%';
    }
    if (widths) {
      return widths;
    }
    // Default staggered widths for visual variety
    const defaultWidths = ['100%', '85%', '70%', '90%', '60%'];
    return defaultWidths[index % defaultWidths.length];
  };

  return (
    <div className={cn('space-y-2', className)} aria-hidden="true">
      {Array.from({ length: lines }, (_, i) => (
        <Skeleton key={i} width={getWidth(i)} height="1em" />
      ))}
    </div>
  );
}

export interface SkeletonCardProps {
  /** Whether to show header */
  hasHeader?: boolean;
  /** Number of content lines */
  contentLines?: number;
  /** Whether to show footer */
  hasFooter?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Skeleton card component for card loading states.
 */
export function SkeletonCard({
  hasHeader = true,
  contentLines = 3,
  hasFooter = true,
  className,
}: SkeletonCardProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800',
        className
      )}
      aria-hidden="true"
    >
      {hasHeader && (
        <div className="mb-4 flex items-center gap-4">
          <Skeleton circle width={40} height={40} />
          <div className="flex-1">
            <Skeleton width="60%" height="1.25em" />
            <Skeleton width="40%" height="0.875em" className="mt-2" />
          </div>
        </div>
      )}
      <SkeletonText lines={contentLines} />
      {hasFooter && (
        <div className="mt-4 flex justify-end gap-2">
          <Skeleton width={80} height={36} />
          <Skeleton width={80} height={36} />
        </div>
      )}
    </div>
  );
}

export interface PageLoaderProps {
  /** Loading message to display */
  message?: string;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Full-page loading overlay component.
 */
export function PageLoader({
  message = 'Loading...',
  className,
}: PageLoaderProps) {
  return (
    <div
      className={cn(
        'flex min-h-[200px] flex-col items-center justify-center gap-4',
        className
      )}
      role="status"
      aria-live="polite"
    >
      <LoadingSpinner size="xl" />
      {message && (
        <p className="text-sm text-gray-500 dark:text-gray-400">{message}</p>
      )}
    </div>
  );
}

export interface DataTableSkeletonProps {
  /** Number of rows to render */
  rows?: number;
  /** Number of columns */
  columns?: number;
  /** Whether to show header */
  hasHeader?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Skeleton for data tables.
 */
export function DataTableSkeleton({
  rows = 5,
  columns = 4,
  hasHeader = true,
  className,
}: DataTableSkeletonProps) {
  return (
    <div
      className={cn(
        'overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700',
        className
      )}
      aria-hidden="true"
    >
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        {hasHeader && (
          <thead className="bg-gray-50 dark:bg-gray-800">
            <tr>
              {Array.from({ length: columns }, (_, i) => (
                <th key={i} className="px-6 py-3">
                  <Skeleton width={`${60 + (i % 3) * 20}%`} height="0.875em" />
                </th>
              ))}
            </tr>
          </thead>
        )}
        <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-900">
          {Array.from({ length: rows }, (_, rowIndex) => (
            <tr key={rowIndex}>
              {Array.from({ length: columns }, (_, colIndex) => (
                <td key={colIndex} className="px-6 py-4">
                  <Skeleton
                    width={`${40 + ((rowIndex + colIndex) % 4) * 15}%`}
                    height="1em"
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
