import { forwardRef, ReactNode } from 'react';
import { cn } from '@/utils/cn';

export interface CardProps {
  /** Title displayed in the card header */
  title?: ReactNode;
  /** Subtitle displayed below the title */
  subtitle?: ReactNode;
  /** Content to display in the footer */
  footer?: ReactNode;
  /** Main content of the card */
  children: ReactNode;
  /** Whether to remove default padding from the body */
  noPadding?: boolean;
  /** Optional actions to display in the header */
  actions?: ReactNode;
  /** Additional CSS classes */
  className?: string;
  /** Click handler */
  onClick?: () => void;
}

/**
 * Card component with header, content, and footer slots.
 * Supports custom styling via className prop.
 */
const Card = forwardRef<HTMLDivElement, CardProps>(
  (
    {
      className,
      title,
      subtitle,
      footer,
      children,
      noPadding = false,
      actions,
      onClick,
    },
    ref
  ) => {
    return (
      <div
        ref={ref}
        className={cn(
          'overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800',
          className
        )}
        onClick={onClick}
      >
        {(title || subtitle || actions) && (
          <div className="border-b border-gray-200 px-6 py-4 dark:border-gray-700">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                {title && (
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                    {title}
                  </h3>
                )}
                {subtitle && (
                  <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {subtitle}
                  </p>
                )}
              </div>
              {actions && (
                <div className="ml-4 flex items-center gap-2">{actions}</div>
              )}
            </div>
          </div>
        )}
        <div
          className={cn(
            noPadding ? '' : 'px-6 py-4',
            'text-gray-700 dark:text-gray-300'
          )}
        >
          {children}
        </div>
        {footer && (
          <div className="border-t border-gray-200 bg-gray-50 px-6 py-4 dark:border-gray-700 dark:bg-gray-900/50">
            {footer}
          </div>
        )}
      </div>
    );
  }
);

Card.displayName = 'Card';

export { Card };
