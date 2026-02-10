import { Menu, Bell, LogOut, User } from 'lucide-react'
import { useAppStore } from '../store/appStore'
import { getCurrentUser, logout } from '../api/auth'
import { toast } from '../stores/toastStore'
import { ProjectSwitcher } from './ProjectSwitcher'

export function Header() {
  const { toggleSidebar } = useAppStore()
  const user = getCurrentUser()

  const handleLogout = () => {
    logout()
    toast.success('Logged out successfully')
  }

  return (
    <header className="flex items-center justify-between h-16 px-6 bg-gradient-to-r from-primary-900 to-primary-800 border-b border-primary-700">
      <div className="flex items-center gap-4">
        <button
          onClick={toggleSidebar}
          className="p-2 text-white/70 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
        >
          <Menu className="w-5 h-5" />
        </button>
        
        {/* Project Switcher */}
        {user && <ProjectSwitcher />}
      </div>

      <div className="flex items-center gap-4">
        <button className="relative p-2 text-white/70 hover:text-white hover:bg-white/10 rounded-lg transition-colors">
          <Bell className="w-5 h-5" />
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-accent-400 rounded-full"></span>
        </button>

        {user && (
          <div className="flex items-center gap-3 pl-4 border-l border-white/20">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 bg-white/20 rounded-full flex items-center justify-center">
                <User className="w-4 h-4 text-white" />
              </div>
              <div className="hidden md:block">
                <p className="text-sm font-medium text-white">{user.display_name}</p>
                <p className="text-xs text-white/70">{user.email}</p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="p-2 text-white/70 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
              title="Logout"
            >
              <LogOut className="w-5 h-5" />
            </button>
          </div>
        )}
      </div>
    </header>
  )
}
