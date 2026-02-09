import { create } from 'zustand';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  /** Unique identifier for the toast */
  id: string;
  /** Toast message */
  message: string;
  /** Type of toast */
  type: ToastType;
  /** Duration in milliseconds (0 for persistent) */
  duration?: number;
  /** Optional action button text */
  actionLabel?: string;
  /** Optional action callback */
  onAction?: () => void;
  /** Whether the toast is dismissible */
  dismissible?: boolean;
}

interface ToastState {
  /** Array of active toasts */
  toasts: Toast[];
  /** Add a new toast */
  addToast: (toast: Omit<Toast, 'id'>) => string;
  /** Remove a toast by ID */
  removeToast: (id: string) => void;
  /** Clear all toasts */
  clearToasts: () => void;
}

/**
 * Zustand store for managing toast notifications.
 * Provides global toast state with add, remove, and clear operations.
 */
export const useToastStore = create<ToastState>((set) => ({
  toasts: [],

  addToast: (toast) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const newToast: Toast = {
      ...toast,
      id,
      duration: toast.duration ?? 5000,
      dismissible: toast.dismissible ?? true,
    };

    set((state) => ({
      toasts: [...state.toasts, newToast],
    }));

    // Auto-dismiss after duration
    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }));
      }, newToast.duration);
    }

    return id;
  },

  removeToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),

  clearToasts: () => set({ toasts: [] }),
}));

/**
 * Helper functions for common toast operations
 */
export const toast = {
  /** Show a success toast */
  success: (message: string, options?: Partial<Omit<Toast, 'id' | 'message' | 'type'>>) =>
    useToastStore.getState().addToast({
      message,
      type: 'success',
      ...options,
    }),

  /** Show an error toast */
  error: (message: string, options?: Partial<Omit<Toast, 'id' | 'message' | 'type'>>) =>
    useToastStore.getState().addToast({
      message,
      type: 'error',
      duration: 0, // Error toasts are persistent by default
      ...options,
    }),

  /** Show a warning toast */
  warning: (message: string, options?: Partial<Omit<Toast, 'id' | 'message' | 'type'>>) =>
    useToastStore.getState().addToast({
      message,
      type: 'warning',
      ...options,
    }),

  /** Show an info toast */
  info: (message: string, options?: Partial<Omit<Toast, 'id' | 'message' | 'type'>>) =>
    useToastStore.getState().addToast({
      message,
      type: 'info',
      ...options,
    }),

  /** Remove a toast by ID */
  dismiss: (id: string) => useToastStore.getState().removeToast(id),

  /** Clear all toasts */
  clear: () => useToastStore.getState().clearToasts(),
};
