import {
  ReactNode,
  useState,
  useMemo,
  MouseEvent,
  TableHTMLAttributes,
} from 'react';
import { cn } from '@/utils/cn';
import { ChevronUp, ChevronDown, ChevronsUpDown } from 'lucide-react';

export type SortDirection = 'asc' | 'desc' | null;

export interface SortConfig<T = Record<string, unknown>> {
  key: keyof T | string | null;
  direction: SortDirection;
}

export interface Column<T = Record<string, unknown>> {
  /** Unique key for the column, should match a property in data items */
  key: keyof T | string;
  /** Header text for the column */
  title: string;
  /** Whether the column is sortable */
  sortable?: boolean;
  /** Custom render function for cell content */
  render?: (value: unknown, item: T, index: number) => ReactNode;
  /** Width of the column (e.g., '100px', '20%') */
  width?: string;
  /** Alignment of the column content */
  align?: 'left' | 'center' | 'right';
  /** Custom class for the column cells */
  className?: string;
}

export interface TableProps<T>
  extends Omit<TableHTMLAttributes<HTMLTableElement>, 'children'> {
  /** Column definitions */
  columns: Column<T>[];
  /** Data to display */
  data: T[];
  /** Function to extract unique key for each row */
  keyExtractor: (item: T, index: number) => string | number;
  /** Whether the table is in a loading state */
  loading?: boolean;
  /** Message to display when there is no data */
  emptyMessage?: ReactNode;
  /** Initial sort configuration */
  initialSort?: SortConfig<T>;
  /** Callback when sort changes */
  onSort?: (config: SortConfig<T>) => void;
  /** Whether to show row hover effect */
  hoverable?: boolean;
  /** Whether rows are clickable */
  onRowClick?: (item: T, index: number) => void;
  /** Custom class for table rows */
  rowClassName?: string | ((item: T, index: number) => string);
  /** Function to generate data-testid for each row */
  rowTestId?: (item: T, index: number) => string;
}

/**
 * Data table component with sorting support.
 * Fully accessible with proper ARIA attributes.
 */
function Table<T = Record<string, unknown>>({
  className,
  columns,
  data,
  keyExtractor,
  loading = false,
  emptyMessage = 'No data available',
  initialSort,
  onSort,
  hoverable = true,
  onRowClick,
  rowClassName,
  rowTestId,
  ...props
}: TableProps<T>) {
  const [sortConfig, setSortConfig] = useState<SortConfig<T>>(
    initialSort || { key: null, direction: null }
  );

  const sortedData = useMemo(() => {
    if (!sortConfig.key || !sortConfig.direction) {
      return data;
    }

    return [...data].sort((a, b) => {
      const aValue = (a as Record<string, unknown>)[sortConfig.key! as string];
      const bValue = (b as Record<string, unknown>)[sortConfig.key! as string];

      if (aValue === bValue) return 0;
      if (aValue === null || aValue === undefined) return 1;
      if (bValue === null || bValue === undefined) return -1;

      const comparison = aValue < bValue ? -1 : 1;
      return sortConfig.direction === 'asc' ? comparison : -comparison;
    });
  }, [data, sortConfig]);

  const handleSort = (key: keyof T | string) => {
    const column = columns.find((c) => c.key === key);
    if (!column?.sortable) return;

    let direction: SortDirection = 'asc';
    if (sortConfig.key === key) {
      if (sortConfig.direction === 'asc') {
        direction = 'desc';
      } else if (sortConfig.direction === 'desc') {
        direction = null;
      }
    }

    const newConfig = { key: direction ? key : null, direction };
    setSortConfig(newConfig);
    onSort?.(newConfig);
  };

  const getSortIcon = (key: keyof T | string) => {
    if (sortConfig.key !== key) {
      return <ChevronsUpDown className="ml-1 h-4 w-4 text-gray-400" />;
    }
    if (sortConfig.direction === 'asc') {
      return <ChevronUp className="ml-1 h-4 w-4 text-blue-600" />;
    }
    return <ChevronDown className="ml-1 h-4 w-4 text-blue-600" />;
  };

  const getAlignment = (align?: 'left' | 'center' | 'right') => {
    switch (align) {
      case 'center':
        return 'text-center';
      case 'right':
        return 'text-right';
      default:
        return 'text-left';
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
      <table
        className={cn('min-w-full divide-y divide-gray-200 dark:divide-gray-700', className)}
        {...props}
      >
        <thead className="bg-gray-50 dark:bg-gray-800">
          <tr>
            {columns.map((column) => (
              <th
                key={String(column.key)}
                scope="col"
                style={{ width: column.width }}
                className={cn(
                  'px-6 py-3 text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400',
                  getAlignment(column.align),
                  column.className,
                  column.sortable && 'cursor-pointer select-none hover:text-gray-700 dark:hover:text-gray-300'
                )}
                onClick={() => handleSort(column.key)}
                aria-sort={
                  sortConfig.key === column.key
                    ? sortConfig.direction === 'asc'
                      ? 'ascending'
                      : 'descending'
                    : 'none'
                }
              >
                <div
                  className={cn(
                    'flex items-center',
                    column.align === 'center' && 'justify-center',
                    column.align === 'right' && 'justify-end'
                  )}
                >
                  {column.title}
                  {column.sortable && getSortIcon(column.key)}
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-900">
          {loading ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-6 py-8 text-center text-sm text-gray-500"
              >
                <div className="flex items-center justify-center">
                  <div className="h-6 w-6 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
                  <span className="ml-2">Loading...</span>
                </div>
              </td>
            </tr>
          ) : sortedData.length === 0 ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-6 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
              >
                {emptyMessage}
              </td>
            </tr>
          ) : (
            sortedData.map((item, index) => (
              <tr
                key={keyExtractor(item, index)}
                data-testid={rowTestId?.(item, index)}
                className={cn(
                  hoverable && 'hover:bg-gray-50 dark:hover:bg-gray-800/50',
                  onRowClick && 'cursor-pointer',
                  typeof rowClassName === 'function'
                    ? rowClassName(item, index)
                    : rowClassName
                )}
                onClick={(_e: MouseEvent<HTMLTableRowElement>) => onRowClick?.(item, index)}
              >
                {columns.map((column) => (
                  <td
                    key={String(column.key)}
                    className={cn(
                      'whitespace-nowrap px-6 py-4 text-sm',
                      getAlignment(column.align),
                      'text-gray-900 dark:text-gray-100',
                      column.className
                    )}
                  >
                    {column.render
                      ? column.render((item as Record<string, unknown>)[column.key as string], item, index)
                      : String((item as Record<string, unknown>)[column.key as string] ?? '-')}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

export { Table };
export type { Column as TableColumn, SortConfig as TableSortConfig, TableProps };
