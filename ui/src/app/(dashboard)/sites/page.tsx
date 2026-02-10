'use client';

import { useQuery } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import { Plus, Globe, ChevronRight, Activity } from 'lucide-react';
import Link from 'next/link';
import { useState } from 'react';
import { CreateSiteModal } from '@/components/CreateSiteModal';
import { cn, relativeTime } from '@/lib/utils';

export default function SitesPage() {
    const client = authStore.getState().client;
    const [isModalOpen, setIsModalOpen] = useState(false);

    const { data, isLoading, error } = useQuery({
        queryKey: ['sites'],
        queryFn: () => client?.listSites(),
        enabled: !!client,
        refetchInterval: 15000,
    });

    if (isLoading) {
        return (
            <div className="flex h-full items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
            </div>
        );
    }

    const sites = data?.sites || [];

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Sites</h1>
                    <p className="text-sm text-slate-500">Manage your edge locations and deployments</p>
                </div>
                <button
                    onClick={() => setIsModalOpen(true)}
                    className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
                >
                    <Plus className="h-4 w-4" />
                    Add Site
                </button>
            </div>

            {sites.length === 0 ? (
                <div className="rounded-xl border-2 border-dashed border-slate-200 bg-white p-12 text-center">
                    <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-50">
                        <Globe className="h-6 w-6 text-slate-400" />
                    </div>
                    <h3 className="mt-4 text-sm font-semibold text-slate-900">No sites found</h3>
                    <p className="mt-1 text-sm text-slate-500">Get started by creating your first edge location.</p>
                    <div className="mt-6">
                        <button
                            onClick={() => setIsModalOpen(true)}
                            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
                        >
                            <Plus className="h-4 w-4" />
                            Add Site
                        </button>
                    </div>
                </div>
            ) : (
                <div className="grid gap-4 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
                    {sites.map((site) => (
                        <Link
                            key={site.id}
                            href={`/sites/${site.id}`}
                            className="group flex flex-col justify-between rounded-xl border bg-white p-6 shadow-sm transition-all hover:border-indigo-300 hover:shadow-md"
                        >
                            <div>
                                <div className="flex items-start justify-between">
                                    <div className="rounded-lg bg-indigo-50 p-2 text-indigo-600 group-hover:bg-indigo-600 group-hover:text-white transition-colors">
                                        <Globe className="h-5 w-5" />
                                    </div>
                                    <span className={cn(
                                        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                                        site.connectivity_state === 'ONLINE'
                                            ? "bg-green-50 text-green-700 ring-1 ring-inset ring-green-600/20"
                                            : "bg-slate-50 text-slate-700 ring-1 ring-inset ring-slate-600/20"
                                    )}>
                                        {site.connectivity_state}
                                    </span>
                                </div>
                                <h3 className="mt-4 text-lg font-semibold text-slate-900">{site.name}</h3>
                                <p className="mt-1 text-xs text-slate-400 font-mono">{site.id}</p>
                            </div>

                            <div className="mt-6 flex items-center justify-between border-t pt-4 text-xs text-slate-500">
                                <div className="flex items-center gap-1">
                                    <Activity className="h-3 w-3" />
                                    <span>Last heartbeat: {relativeTime(site.last_heartbeat_at || '')}</span>
                                </div>
                                <ChevronRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
}
