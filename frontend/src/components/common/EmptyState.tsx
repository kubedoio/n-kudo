import { ReactNode } from 'react';
import { cn } from '@/utils/cn';
import { Button } from './Button';
import {
  Inbox,
  Search,
  FolderOpen,
  FileX,
  AlertCircle,
  LucideIcon,
} from 'lucide-react';

export type EmptyStateIcon =
  | 'inbox'
  | 'search'
  | 'folder'
  | 'file'
  | 'error'
  | 'custom';

export interface EmptyStateProps {
  /** Title of the empty state */
  title: string;
  /** Description text */
  description?: string;
  /** Icon to display (preset name or custom element) */
  icon?: EmptyStateIcon | ReactNode;
  /** Preset icon size */
  iconSize?: 'sm' | 'md' | 'lg' | 'xl';
  /** Primary action button */
  action?: {
    label: string;
    onClick: () => void;
    icon?: ReactNode;
  };
  /** Secondary action button */
  secondaryAction?: {
    label: string;
    onClick: () => void;
    icon?: ReactNode;
  };
  /** Additional CSS classes */
  className?: string;
}

const iconComponents: Record<string, LucideIcon> = {
  inbox: Inbox,
  search: Search,
  folder: FolderOpen,
  file: FileX,
  error: AlertCircle,
};

const iconSizes = {
  sm: 'h-8 w-8',
  md: 'h-12 w-12',
  lg: 'h-16 w-16',
  xl: 'h-24 w-24',
};

const iconContainerSizes = {
  sm: 'p-2',
  md: 'p-3',
  lg: 'p-4',
  xl: 'p-6',
};

/**
 * Empty state component for displaying when there's no data or content.
 * Includes optional icon, title, description, and action buttons.
 */
export function EmptyState({
  title,
  description,
  icon = 'inbox',
  iconSize = 'lg',
  action,
  secondaryAction,
  className,
}: EmptyStateProps) {
  const renderIcon = () => {
    if (icon === 'custom') {
      return null;
    }

    if (typeof icon === 'string' && icon in iconComponents) {
      const IconComponent = iconComponents[icon];
      return (
        <IconComponent
          className={cn('text-gray-400 dark:text-gray-500', iconSizes[iconSize])}
          aria-hidden="true"
        />
      );
    }

    return (
      <div className={cn('text-gray-400 dark:text-gray-500', iconSizes[iconSize])}>
        {icon}
      </div>
    );
  };

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        className
      )}
      role="status"
      aria-live="polite"
    >
      {/* Icon Container */}
      {icon !== 'custom' && (
        <div
          className={cn(
            'mb-4 rounded-full bg-gray-100 dark:bg-gray-800',
            iconContainerSizes[iconSize]
          )}
        >
          {renderIcon()}
        </div>
      )}

      {/* Title */}
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
        {title}
      </h3>

      {/* Description */}
      {description && (
        <p className="mt-1 max-w-sm text-sm text-gray-500 dark:text-gray-400">
          {description}
        </p>
      )}

      {/* Actions */}
      {(action || secondaryAction) && (
        <div className="mt-6 flex items-center gap-3">
          {action && (
            <Button
              onClick={action.onClick}
              leftIcon={action.icon}
              variant="primary"
            >
              {action.label}
            </Button>
          )}
          {secondaryAction && (
            <Button
              onClick={secondaryAction.onClick}
              leftIcon={secondaryAction.icon}
              variant="secondary"
            >
              {secondaryAction.label}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

export interface EmptySearchResultsProps extends Omit<EmptyStateProps, 'icon'> {
  /** Search query that returned no results */
  query?: string;
  /** Callback to clear the search */
  onClearSearch?: () => void;
}

/**
 * Specialized empty state for search results with no matches.
 */
export function EmptySearchResults({
  query,
  onClearSearch,
  title: titleProp,
  description: descriptionProp,
  ...props
}: EmptySearchResultsProps) {
  const title = titleProp || (query ? `No results for "${query}"` : 'No results found');
  const description =
    descriptionProp ||
    'Try adjusting your search or filters to find what you\'re looking for.';

  return (
    <EmptyState
      icon="search"
      title={title}
      description={description}
      secondaryAction={
        onClearSearch
          ? {
              label: 'Clear search',
              onClick: onClearSearch,
            }
          : undefined
      }
      {...props}
    />
  );
}

export interface EmptyTableProps {
  /** Message to display */
  message?: string;
  /** Description text */
  description?: string;
  /** Create action */
  onCreate?: () => void;
  /** Create button label */
  createLabel?: string;
  /** Number of columns (for colspan) */
  colSpan?: number;
}

/**
 * Empty state specifically designed for table cells.
 */
export function EmptyTable({
  message = 'No data available',
  description,
  onCreate,
  createLabel = 'Create new',
  colSpan = 1,
}: EmptyTableProps) {
  return (
    <tr>
      <td colSpan={colSpan} className="px-6 py-12">
        <EmptyState
          icon="inbox"
          iconSize="md"
          title={message}
          description={description}
          action={
            onCreate
              ? {
                  label: createLabel,
                  onClick: onCreate,
                }
              : undefined
          }
        />
      </td>
    </tr>
  );
}
