import { forwardRef, SelectHTMLAttributes } from 'react';
import { cn } from '@/utils/cn';
import { ChevronDown, AlertCircle } from 'lucide-react';

export interface SelectOption {
  /** Unique value for the option */
  value: string;
  /** Display label for the option */
  label: string;
  /** Whether the option is disabled */
  disabled?: boolean;
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  /** Label text for the select */
  label?: string;
  /** Array of options to display */
  options: SelectOption[];
  /** Error message to display */
  error?: string;
  /** Helper text displayed below the select */
  helperText?: string;
  /** Placeholder text for the default empty option */
  placeholder?: string;
}

/**
 * Dropdown select component with label and error message support.
 * Fully accessible with proper ARIA attributes.
 */
const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      className,
      label,
      options,
      error,
      helperText,
      placeholder,
      disabled,
      required,
      id,
      ...props
    },
    ref
  ) => {
    const selectId = id || `select-${Math.random().toString(36).substr(2, 9)}`;
    const errorId = error ? `${selectId}-error` : undefined;
    const helperId = helperText ? `${selectId}-helper` : undefined;
    const describedBy = [helperId, errorId].filter(Boolean).join(' ') || undefined;

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={selectId}
            className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {label}
            {required && (
              <span className="ml-1 text-red-500" aria-hidden="true">
                *
              </span>
            )}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={selectId}
            disabled={disabled}
            required={required}
            aria-invalid={error ? 'true' : 'false'}
            aria-describedby={describedBy}
            aria-required={required}
            className={cn(
              'block w-full appearance-none rounded-md border px-3 py-2 pr-10 text-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-0 disabled:cursor-not-allowed disabled:bg-gray-100 disabled:opacity-50 dark:bg-gray-800 dark:text-white',
              error
                ? 'border-red-300 focus:border-red-500 focus:ring-red-500 dark:border-red-600'
                : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600',
              className
            )}
            {...props}
          >
            {placeholder && (
              <option value="" disabled>
                {placeholder}
              </option>
            )}
            {options.map((option) => (
              <option
                key={option.value}
                value={option.value}
                disabled={option.disabled}
              >
                {option.label}
              </option>
            ))}
          </select>
          <div
            className={cn(
              'pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3',
              error ? 'text-red-500' : 'text-gray-400'
            )}
          >
            {error ? (
              <AlertCircle className="h-5 w-5" aria-hidden="true" />
            ) : (
              <ChevronDown className="h-5 w-5" aria-hidden="true" />
            )}
          </div>
        </div>
        {helperText && !error && (
          <p
            id={helperId}
            className="mt-1.5 text-sm text-gray-500 dark:text-gray-400"
          >
            {helperText}
          </p>
        )}
        {error && (
          <p
            id={errorId}
            className="mt-1.5 text-sm text-red-600 dark:text-red-400"
            role="alert"
          >
            {error}
          </p>
        )}
      </div>
    );
  }
);

Select.displayName = 'Select';

export { Select };
