'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { authStore } from '@/store/auth';
import {
    BarChart3,
    Box,
    LayoutDashboard,
    Activity,
    Settings,
    LogOut,
    Cloud,
    Globe
} from 'lucide-react';
import { cn } from '@/lib/utils';

const navItems = [
    { icon: LayoutDashboard, label: 'Sites', href: '/sites' },
    { icon: Globe, label: 'Hosts', href: '/hosts' },
    { icon: Box, label: 'VMs', href: '/vms' },
    { icon: Activity, label: 'Executions', href: '/executions' },
    { icon: Settings, label: 'Settings', href: '/settings' },
];

export function Sidebar() {
    const pathname = usePathname();
    const router = useRouter();
    const config = authStore.getState().config;

    const handleLogout = () => {
        authStore.setConfig(null);
        router.push('/');
    };

    return (
        <div className="flex h-screen w-64 flex-col border-r bg-white shadow-sm">
            <div className="flex h-16 items-center border-b px-6">
                <Link href="/sites" className="flex items-center gap-2">
                    <Cloud className="h-6 w-6 text-indigo-600" />
                    <span className="text-xl font-bold text-slate-900 tracking-tight">nkudo</span>
                </Link>
            </div>

            <nav className="flex-1 space-y-1 p-4">
                {navItems.map((item) => {
                    const isActive = pathname.startsWith(item.href);
                    return (
                        <Link
                            key={item.href}
                            href={item.href}
                            className={cn(
                                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                                isActive
                                    ? 'bg-indigo-50 text-indigo-600'
                                    : 'text-slate-600 hover:bg-slate-50 hover:text-indigo-600'
                            )}
                        >
                            <item.icon className="h-5 w-5" />
                            {item.label}
                        </Link>
                    );
                })}
            </nav>

            <div className="border-t p-4">
                <div className="mb-4 rounded-lg bg-slate-50 p-3">
                    <p className="text-[10px] uppercase tracking-wider text-slate-500 font-bold">Tenant</p>
                    <p className="truncate text-xs font-medium text-slate-700">{config?.tenantId || 'Unknown'}</p>
                </div>
                <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-red-50 hover:text-red-600"
                >
                    <LogOut className="h-5 w-5" />
                    Logout
                </button>
            </div>
        </div>
    );
}
