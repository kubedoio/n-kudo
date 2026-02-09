import { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { cn } from '@/utils/cn';
import { ChevronRight, Home } from 'lucide-react';

export interface Breadcrumb {
  /** Display label */
  label: string;
  /** Route path (optional - if not provided, item is not clickable) */
  path?: string;
  /** Icon to display before label */
  icon?: ReactNode;
}

export interface HeaderUser {
  /** User's display name */
  name: string;
  /** User's email */
  email?: string;
  /** Avatar URL or fallback element */
  avatar?: string | ReactNode;
  /** User role/title */
  role?: string;
}

export interface HeaderProps {
  /** Page title */
  title: ReactNode;
  /** Optional subtitle */
  subtitle?: ReactNode;
  /** Breadcrumb items */
  breadcrumbs?: Breadcrumb[];
  /** User information */
  user?: HeaderUser;
  /** Actions to display on the right */
  actions?: ReactNode;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Top header component with breadcrumbs, title, and user info.
 * Fully accessible with proper ARIA attributes.
 */
export function Header({
  title,
  subtitle,
  breadcrumbs,
  user,
  actions,
  className,
}: HeaderProps) {
  return (
    <header
      className={cn(
        'border-b border-gray-200 bg-white px-4 py-4 dark:border-gray-700 dark:bg-gray-900 sm:px-6 lg:px-8',
        className
      )}
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex-1">
          {/* Breadcrumbs */}
          {breadcrumbs && breadcrumbs.length > 0 && (
            <nav
              className="mb-3 flex items-center text-sm text-gray-500 dark:text-gray-400"
              aria-label="Breadcrumb"
            >
              <ol className="flex flex-wrap items-center gap-1">
                <li>
                  <Link
                    to="/"
                    className="flex items-center hover:text-gray-700 dark:hover:text-gray-300"
                    aria-label="Home"
                  >
                    <Home className="h-4 w-4" />
                  </Link>
                </li>
                {breadcrumbs.map((crumb, index) => (
                  <li key={index} className="flex items-center">
                    <ChevronRight
                      className="mx-1 h-4 w-4 flex-shrink-0"
                      aria-hidden="true"
                    />
                    {crumb.path ? (
                      <Link
                        to={crumb.path}
                        className={cn(
                          'hover:text-gray-700 dark:hover:text-gray-300',
                          index === breadcrumbs.length - 1 &&
                            'font-medium text-gray-900 dark:text-white'
                        )}
                        aria-current={
                          index === breadcrumbs.length - 1 ? 'page' : undefined
                        }
                      >
                        <span className="flex items-center gap-1">
                          {crumb.icon && (
                            <span className="h-4 w-4">{crumb.icon}</span>
                          )}
                          {crumb.label}
                        </span>
                      </Link>
                    ) : (
                      <span
                        className="flex items-center gap-1 font-medium text-gray-900 dark:text-white"
                        aria-current="page"
                      >
                        {crumb.icon && (
                          <span className="h-4 w-4">{crumb.icon}</span>
                        )}
                        {crumb.label}
                      </span>
                    )}
                  </li>
                ))}
              </ol>
            </nav>
          )}

          {/* Title and Subtitle */}
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {title}
            </h1>
            {subtitle && (
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {subtitle}
              </p>
            )}
          </div>
        </div>

        {/* Right Section: Actions and User */}
        <div className="flex items-center gap-4">
          {actions && (
            <div className="flex items-center gap-2">{actions}</div>
          )}

          {user && (
            <div className="flex items-center gap-3 border-l border-gray-200 pl-4 dark:border-gray-700">
              <div className="hidden text-right sm:block">
                <p className="text-sm font-medium text-gray-900 dark:text-white">
                  {user.name}
                </p>
                {user.role && (
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {user.role}
                  </p>
                )}
              </div>
              <div className="h-9 w-9 flex-shrink-0 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                {typeof user.avatar === 'string' ? (
                  <img
                    src={user.avatar}
                    alt={`${user.name}'s avatar`}
                    className="h-full w-full object-cover"
                  />
                ) : user.avatar ? (
                  user.avatar
                ) : (
                  <div className="flex h-full w-full items-center justify-center text-sm font-medium text-gray-500 dark:text-gray-400">
                    {user.name.charAt(0).toUpperCase()}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
