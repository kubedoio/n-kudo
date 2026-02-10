import { NavLink, useParams } from 'react-router-dom'
import {
  FolderOpen,
  Layout,
  Server,
  Settings,
  Shield,
} from 'lucide-react'
import { useAppStore } from '../store/appStore'
import { cn } from '../lib/utils'

export function Sidebar() {
  const { sidebarOpen } = useAppStore()
  const { projectId } = useParams()

  const navItems = [
    { path: '/projects', label: 'Projects', icon: FolderOpen },
  ]

  const projectNavItems = projectId
    ? [
        { path: `/projects/${projectId}`, label: 'Overview', icon: Layout },
        { path: `/projects/${projectId}/sites`, label: 'Sites', icon: Server },
        { path: `/projects/${projectId}/settings`, label: 'Settings', icon: Settings },
      ]
    : []

  return (
    <aside
      className={cn(
        'flex flex-col bg-primary-900 transition-all duration-300',
        sidebarOpen ? 'w-64' : 'w-20'
      )}
    >
      {/* Logo */}
      <div className="flex items-center h-16 px-4 border-b border-primary-800">
        <div className="flex items-center justify-center w-10 h-10 bg-accent-600 rounded-lg">
          <Shield className="w-6 h-6 text-white" />
        </div>
        {sidebarOpen && (
          <span className="ml-3 text-lg font-semibold text-white">N-Kudo</span>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        <div className="mb-6">
          {sidebarOpen && (
            <p className="px-4 mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
              Main
            </p>
          )}
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                cn(
                  'sidebar-link',
                  isActive && 'active',
                  !sidebarOpen && 'justify-center px-2'
                )
              }
              title={!sidebarOpen ? item.label : undefined}
            >
              <item.icon className="w-5 h-5 shrink-0" />
              {sidebarOpen && <span>{item.label}</span>}
            </NavLink>
          ))}
        </div>

        {projectId && (
          <div>
            {sidebarOpen && (
              <p className="px-4 mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                Project
              </p>
            )}
            {projectNavItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  cn(
                    'sidebar-link',
                    isActive && 'active',
                    !sidebarOpen && 'justify-center px-2'
                  )
                }
                title={!sidebarOpen ? item.label : undefined}
              >
                <item.icon className="w-5 h-5 shrink-0" />
                {sidebarOpen && <span>{item.label}</span>}
              </NavLink>
            ))}
          </div>
        )}
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-primary-800">
        <div className={cn('flex items-center', !sidebarOpen && 'justify-center')}>
          <div className="w-8 h-8 bg-primary-700 rounded-full flex items-center justify-center">
            <span className="text-xs font-medium text-white">A</span>
          </div>
          {sidebarOpen && (
            <div className="ml-3">
              <p className="text-sm font-medium text-white">Admin User</p>
              <p className="text-xs text-gray-400">admin@nkudo.io</p>
            </div>
          )}
        </div>
      </div>
    </aside>
  )
}
