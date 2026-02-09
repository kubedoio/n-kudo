import {
  useEffect,
  useRef,
  useCallback,
  ReactNode,
  useState,
} from 'react';
import { createPortal } from 'react-dom';
import { cn } from '@/utils/cn';
import { X } from 'lucide-react';
import { Button } from './Button';

export interface ModalProps {
  /** Whether the modal is open */
  isOpen: boolean;
  /** Callback when the modal should close */
  onClose: () => void;
  /** Modal title */
  title?: ReactNode;
  /** Modal description/subtitle */
  description?: ReactNode;
  /** Content to display in the modal body */
  children: ReactNode;
  /** Content to display in the modal footer */
  footer?: ReactNode;
  /** Size of the modal */
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
  /** Whether to show the close button in the header */
  showCloseButton?: boolean;
  /** Whether pressing Escape should close the modal */
  closeOnEscape?: boolean;
  /** Whether clicking the backdrop should close the modal */
  closeOnBackdropClick?: boolean;
  /** Additional CSS classes for the modal panel */
  className?: string;
  /** ID for accessibility (auto-generated if not provided) */
  id?: string;
}

const sizes = {
  sm: 'max-w-md',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
  full: 'max-w-full mx-4',
};

/**
 * Modal/Dialog component with backdrop, animations, and accessibility support.
 * Features focus trapping, keyboard navigation, and ARIA attributes.
 */
export function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  size = 'md',
  showCloseButton = true,
  closeOnEscape = true,
  closeOnBackdropClick = true,
  className,
  id,
}: ModalProps) {
  const [isClosing, setIsClosing] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false);
  const modalRef = useRef<HTMLDivElement>(null);
  const titleId = id ? `${id}-title` : `modal-title-${Math.random().toString(36).substr(2, 9)}`;
  const descriptionId = description
    ? id
      ? `${id}-description`
      : `modal-description-${Math.random().toString(36).substr(2, 9)}`
    : undefined;

  // Handle closing with animation
  const handleClose = useCallback(() => {
    setIsClosing(true);
    setTimeout(() => {
      setIsClosing(false);
      setIsAnimating(false);
      onClose();
    }, 200); // Match transition duration
  }, [onClose]);

  // Trap focus within modal
  const trapFocus = useCallback((e: KeyboardEvent) => {
    if (e.key !== 'Tab' || !modalRef.current) return;

    const focusableElements = modalRef.current.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    if (e.shiftKey) {
      if (document.activeElement === firstElement) {
        e.preventDefault();
        lastElement?.focus();
      }
    } else {
      if (document.activeElement === lastElement) {
        e.preventDefault();
        firstElement?.focus();
      }
    }
  }, []);

  // Handle escape key
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape' && closeOnEscape) {
        handleClose();
      } else if (e.key === 'Tab') {
        trapFocus(e);
      }
    },
    [closeOnEscape, handleClose, trapFocus]
  );

  // Store previously focused element and restore on close
  const previousFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (isOpen) {
      previousFocusRef.current = document.activeElement as HTMLElement;
      setIsAnimating(true);
      // Focus the modal after a short delay for animation
      setTimeout(() => {
        modalRef.current?.focus();
      }, 50);
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
      previousFocusRef.current?.focus();
    };
  }, [isOpen, handleKeyDown]);

  // Handle backdrop click
  const handleBackdropClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === e.currentTarget && closeOnBackdropClick) {
        handleClose();
      }
    },
    [closeOnBackdropClick, handleClose]
  );

  if (!isOpen && !isClosing) return null;

  const modalContent = (
    <div
      className="fixed inset-0 z-50"
      aria-labelledby={title ? titleId : undefined}
      aria-describedby={descriptionId}
      role="dialog"
      aria-modal="true"
    >
      {/* Backdrop */}
      <div
        className={cn(
          'fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity duration-200',
          isAnimating && !isClosing ? 'opacity-100' : 'opacity-0'
        )}
        onClick={handleBackdropClick}
        aria-hidden="true"
      />

      {/* Modal container */}
      <div
        className="fixed inset-0 overflow-y-auto"
        onClick={handleBackdropClick}
      >
        <div className="flex min-h-full items-center justify-center p-4 text-center">
          <div
            ref={modalRef}
            tabIndex={-1}
            className={cn(
              'relative w-full transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all dark:bg-gray-800',
              sizes[size],
              isAnimating && !isClosing
                ? 'opacity-100 scale-100 translate-y-0'
                : 'opacity-0 scale-95 translate-y-4',
              'duration-200 ease-out',
              className
            )}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            {(title || showCloseButton) && (
              <div className="flex items-start justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
                <div className="flex-1 pr-4">
                  {title && (
                    <h3
                      id={titleId}
                      className="text-lg font-semibold leading-6 text-gray-900 dark:text-white"
                    >
                      {title}
                    </h3>
                  )}
                  {description && (
                    <p
                      id={descriptionId}
                      className="mt-1 text-sm text-gray-500 dark:text-gray-400"
                    >
                      {description}
                    </p>
                  )}
                </div>
                {showCloseButton && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleClose}
                    className="-mr-2 flex-shrink-0"
                    aria-label="Close modal"
                  >
                    <X className="h-5 w-5" />
                  </Button>
                )}
              </div>
            )}

            {/* Body */}
            <div className="px-6 py-4">
              {!title && description && (
                <p
                  id={descriptionId}
                  className="mb-4 text-sm text-gray-500 dark:text-gray-400"
                >
                  {description}
                </p>
              )}
              {children}
            </div>

            {/* Footer */}
            {footer && (
              <div className="border-t border-gray-200 bg-gray-50 px-6 py-4 dark:border-gray-700 dark:bg-gray-900/50">
                {footer}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );

  return createPortal(modalContent, document.body);
}

export interface ConfirmModalProps extends Omit<ModalProps, 'footer'> {
  /** Text for the confirm button */
  confirmLabel?: string;
  /** Text for the cancel button */
  cancelLabel?: string;
  /** Callback when confirmed */
  onConfirm: () => void;
  /** Variant of the confirm button */
  confirmVariant?: 'primary' | 'danger';
  /** Whether the confirm action is loading */
  isLoading?: boolean;
}

/**
 * Pre-configured confirmation modal with confirm/cancel buttons.
 */
export function ConfirmModal({
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  onConfirm,
  confirmVariant = 'primary',
  isLoading,
  children,
  onClose,
  ...props
}: ConfirmModalProps) {
  const handleConfirm = () => {
    onConfirm();
  };

  const footer = (
    <div className="flex justify-end gap-3">
      <Button variant="secondary" onClick={onClose} disabled={isLoading}>
        {cancelLabel}
      </Button>
      <Button
        variant={confirmVariant}
        onClick={handleConfirm}
        loading={isLoading}
      >
        {confirmLabel}
      </Button>
    </div>
  );

  return (
    <Modal onClose={onClose} footer={footer} {...props}>
      {children}
    </Modal>
  );
}
