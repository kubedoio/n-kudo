import { NavLink, Outlet, useLocation } from 'react-router-dom'
import {
    LayoutDashboard,
    Bell,
    User,
    Palette,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { TooltipProvider } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

export default function AppShell() {
    const location = useLocation()
    const isDashboard = location.pathname === '/'

    return (
        <TooltipProvider>
            <div className="flex min-h-screen flex-col">
                {/* Header */}
                <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                    <div className="flex h-14 items-center justify-between px-4">
                        {/* Left: Branding */}
                        <div className="flex items-center gap-3">
                            <NavLink to="/" className="flex items-center gap-2">
                                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground font-bold text-sm">
                                    nK
                                </div>
                                <span className="hidden font-semibold sm:inline-block">n-kudo</span>
                            </NavLink>
                            <span className="text-muted-foreground text-sm hidden md:inline">
                                {isDashboard ? 'DASHBOARD – Overview, choose for details' : getPageLabel(location.pathname)}
                            </span>
                        </div>

                        {/* Right: Nav Buttons */}
                        <nav className="flex items-center gap-1" aria-label="Main navigation">
                            <NavLink to="/">
                                {({ isActive }) => (
                                    <Button variant={isActive ? 'secondary' : 'ghost'} size="sm" aria-label="Dashboard">
                                        <LayoutDashboard className="mr-1.5 h-4 w-4" />
                                        <span className="hidden sm:inline">Dashboard</span>
                                    </Button>
                                )}
                            </NavLink>

                            <Button variant="ghost" size="sm" disabled className="relative" aria-label="Designer (MVP-2)">
                                <Palette className="mr-1.5 h-4 w-4" />
                                <span className="hidden sm:inline">Designer</span>
                                <Badge variant="outline" className="ml-1.5 text-[10px] px-1.5 py-0">MVP-2</Badge>
                            </Button>

                            <NavLink to="/notifications">
                                {({ isActive }) => (
                                    <Button variant={isActive ? 'secondary' : 'ghost'} size="sm" aria-label="Notifications">
                                        <Bell className="h-4 w-4" />
                                    </Button>
                                )}
                            </NavLink>

                            <NavLink to="/profile">
                                {({ isActive }) => (
                                    <Button variant={isActive ? 'secondary' : 'ghost'} size="sm" aria-label="Profile & Settings">
                                        <User className="h-4 w-4" />
                                    </Button>
                                )}
                            </NavLink>
                        </nav>
                    </div>
                </header>

                {/* Main content */}
                <main className="flex-1 p-4">
                    <Outlet />
                </main>
            </div>
        </TooltipProvider>
    )
}

function getPageLabel(pathname: string): string {
    if (pathname.startsWith('/sites/')) return 'SITE DETAIL'
    if (pathname.startsWith('/executions/')) return 'EXECUTION LOGS'
    if (pathname === '/enroll') return 'ENROLLMENT HELPER'
    if (pathname === '/notifications') return 'NOTIFICATIONS'
    if (pathname === '/profile') return 'PROFILE & SETTINGS'
    return 'DASHBOARD'
}
