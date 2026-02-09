import { useCallback } from 'react';
import { cn } from '@/utils/cn';
import { useToastStore, Toast as ToastType } from '@/stores/toastStore';
import {
  CheckCircle,
  XCircle,
  AlertTriangle,
  Info,
  X,
} from 'lucide-react';
import { Button } from './Button';

interface ToastItemProps {
  toast: ToastType;
  onRemove: (id: string) => void;
}

const toastIcons = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
};

const toastStyles = {
  success:
    'bg-green-50 border-green-200 text-green-800 dark:bg-green-900/20 dark:border-green-800 dark:text-green-300',
  error:
    'bg-red-50 border-red-200 text-red-800 dark:bg-red-900/20 dark:border-red-800 dark:text-red-300',
  warning:
    'bg-yellow-50 border-yellow-200 text-yellow-800 dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-300',
  info: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-300',
};

const iconStyles = {
  success: 'text-green-500 dark:text-green-400',
  error: 'text-red-500 dark:text-red-400',
  warning: 'text-yellow-500 dark:text-yellow-400',
  info: 'text-blue-500 dark:text-blue-400',
};

function ToastItem({ toast, onRemove }: ToastItemProps) {
  const Icon = toastIcons[toast.type];

  const handleAction = useCallback(() => {
    toast.onAction?.();
    onRemove(toast.id);
  }, [toast, onRemove]);

  const handleDismiss = useCallback(() => {
    onRemove(toast.id);
  }, [toast.id, onRemove]);

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(
        'pointer-events-auto w-full max-w-sm overflow-hidden rounded-lg border shadow-lg transition-all',
        'animate-in slide-in-from-right-full fade-in duration-300',
        toastStyles[toast.type]
      )}
    >
      <div className="p-4">
        <div className="flex items-start gap-3">
          <div className={cn('flex-shrink-0', iconStyles[toast.type])}>
            <Icon className="h-5 w-5" aria-hidden="true" />
          </div>
          <div className="flex-1 pt-0.5">
            <p className="text-sm font-medium">{toast.message}</p>
            {toast.actionLabel && (
              <div className="mt-3">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleAction}
                  className={cn(
                    'h-auto py-1 px-2 text-xs',
                    toast.type === 'error' &&
                      'text-red-700 hover:text-red-800 hover:bg-red-100 dark:text-red-300 dark:hover:bg-red-900/30',
                    toast.type === 'success' &&
                      'text-green-700 hover:text-green-800 hover:bg-green-100 dark:text-green-300 dark:hover:bg-green-900/30',
                    toast.type === 'warning' &&
                      'text-yellow-700 hover:text-yellow-800 hover:bg-yellow-100 dark:text-yellow-300 dark:hover:bg-yellow-900/30',
                    toast.type === 'info' &&
                      'text-blue-700 hover:text-blue-800 hover:bg-blue-100 dark:text-blue-300 dark:hover:bg-blue-900/30'
                  )}
                >
                  {toast.actionLabel}
                </Button>
              </div>
            )}
          </div>
          {toast.dismissible && (
            <button
              onClick={handleDismiss}
              className="flex-shrink-0 rounded-md p-1 opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-offset-0"
              aria-label="Dismiss notification"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

export interface ToastContainerProps {
  /** Position of the toast container on screen */
  position?:
    | 'top-left'
    | 'top-center'
    | 'top-right'
    | 'bottom-left'
    | 'bottom-center'
    | 'bottom-right';
  /** Maximum number of toasts to display at once */
  limit?: number;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Toast container component that displays notifications using Zustand store.
 * Supports multiple positions and automatic dismissal.
 */
export function ToastContainer({
  position = 'top-right',
  limit = 5,
  className,
}: ToastContainerProps) {
  const { toasts, removeToast } = useToastStore();

  const positions = {
    'top-left': 'top-0 left-0',
    'top-center': 'top-0 left-1/2 -translate-x-1/2',
    'top-right': 'top-0 right-0',
    'bottom-left': 'bottom-0 left-0',
    'bottom-center': 'bottom-0 left-1/2 -translate-x-1/2',
    'bottom-right': 'bottom-0 right-0',
  };

  const visibleToasts = toasts.slice(-limit);

  return (
    <div
      className={cn(
        'fixed z-50 flex flex-col gap-2 p-4 sm:p-6',
        positions[position],
        position.startsWith('top') ? 'flex-col-reverse' : 'flex-col',
        className
      )}
      aria-live="polite"
      aria-atomic="true"
    >
      {visibleToasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onRemove={removeToast} />
      ))}
    </div>
  );
}

// Re-export toast helper functions
export { toast } from '@/stores/toastStore';
