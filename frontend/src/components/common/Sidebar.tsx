import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { cn } from '@/utils/cn';

export interface SidebarItem {
  /** Unique identifier for the item */
  id: string;
  /** Display label */
  label: string;
  /** Route path */
  path: string;
  /** Icon component */
  icon: ReactNode;
  /** Whether the item is disabled */
  disabled?: boolean;
  /** Badge text to display */
  badge?: string | number;
  /** Nested child items */
  children?: Omit<SidebarItem, 'children'>[];
}

export interface SidebarProps {
  /** Logo or brand element to display at the top */
  logo?: ReactNode;
  /** Navigation items */
  items: SidebarItem[];
  /** Currently active item ID */
  activeItem?: string;
  /** Footer content (user info, logout, etc.) */
  footer?: ReactNode;
  /** Callback when an item is clicked */
  onItemClick?: (item: SidebarItem) => void;
  /** Whether the sidebar is collapsed */
  collapsed?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Navigation sidebar component with logo, menu items, and footer.
 * Fully accessible with proper ARIA attributes.
 */
export function Sidebar({
  logo,
  items,
  activeItem,
  footer,
  onItemClick,
  collapsed = false,
  className,
}: SidebarProps) {
  const location = useLocation();

  const isActive = (item: SidebarItem): boolean => {
    if (activeItem) {
      return activeItem === item.id;
    }
    return location.pathname === item.path || location.pathname.startsWith(item.path + '/');
  };

  return (
    <aside
      className={cn(
        'flex h-full flex-col border-r border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900',
        collapsed ? 'w-16' : 'w-64',
        className
      )}
      aria-label="Main navigation"
    >
      {/* Logo Section */}
      {logo && (
        <div className="flex h-16 items-center border-b border-gray-200 px-4 dark:border-gray-700">
          {logo}
        </div>
      )}

      {/* Navigation Items */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <ul className="space-y-1" role="menu">
          {items.map((item) => {
            const active = isActive(item);
            return (
              <li key={item.id} role="none">
                <Link
                  to={item.path}
                  role="menuitem"
                  aria-current={active ? 'page' : undefined}
                  aria-disabled={item.disabled}
                  onClick={(e) => {
                    if (item.disabled) {
                      e.preventDefault();
                      return;
                    }
                    onItemClick?.(item);
                  }}
                  className={cn(
                    'group flex items-center rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    active
                      ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400'
                      : 'text-gray-700 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:text-white',
                    item.disabled && 'cursor-not-allowed opacity-50',
                    collapsed && 'justify-center px-2'
                  )}
                >
                  <span
                    className={cn(
                      'flex-shrink-0',
                      active
                        ? 'text-blue-600 dark:text-blue-400'
                        : 'text-gray-400 group-hover:text-gray-500 dark:text-gray-500 dark:group-hover:text-gray-400',
                      collapsed ? 'h-6 w-6' : 'mr-3 h-5 w-5'
                    )}
                    aria-hidden="true"
                  >
                    {item.icon}
                  </span>
                  {!collapsed && (
                    <>
                      <span className="flex-1 truncate">{item.label}</span>
                      {item.badge !== undefined && (
                        <span
                          className={cn(
                            'ml-3 inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                            active
                              ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
                              : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300'
                          )}
                        >
                          {item.badge}
                        </span>
                      )}
                    </>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Footer Section */}
      {footer && (
        <div className="border-t border-gray-200 p-4 dark:border-gray-700">
          {footer}
        </div>
      )}
    </aside>
  );
}
