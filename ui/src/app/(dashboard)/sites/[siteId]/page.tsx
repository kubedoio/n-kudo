'use client';

import { useState, use } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import {
    Server,
    Cpu,
    HardDrive,
    Terminal,
    Plus,
    Globe,
    Activity,
    Copy,
    Check,
    ShieldCheck,
    Zap,
    Box,
    LayoutDashboard as LayoutDashboardIcon,
    Loader2 as Loader2Icon,
    AlertCircle as AlertCircleIcon
} from 'lucide-react';
import { CreateVMWizard } from '@/components/CreateVMWizard';
import { cn, relativeTime, formatBytes } from '@/lib/utils';

export default function SiteDetailPage({ params: paramsPromise }: { params: Promise<{ siteId: string }> }) {
    const params = use(paramsPromise);
    const siteId = params.siteId;
    const client = authStore.getState().client;
    const queryClient = useQueryClient();
    const [activeTab, setActiveTab] = useState('overview');
    const [token, setToken] = useState<string | null>(null);
    const [copied, setCopied] = useState(false);
    const [isWizardOpen, setIsWizardOpen] = useState(false);

    const { data: sitesData } = useQuery({
        queryKey: ['sites'],
        queryFn: () => client?.listSites(),
        enabled: !!client,
    });

    const site = sitesData?.sites.find(s => s.id === siteId);

    const { data: hostsData } = useQuery({
        queryKey: ['hosts', siteId],
        queryFn: () => client?.listHosts(siteId),
        enabled: !!client,
        refetchInterval: 5000,
    });

    const { data: vmsData } = useQuery({
        queryKey: ['vms', siteId],
        queryFn: () => client?.listVMs(siteId),
        enabled: !!client,
        refetchInterval: 5000,
    });

    const createTokenMutation = useMutation({
        mutationFn: () => client!.createEnrollmentToken(siteId),
        onSuccess: (data) => {
            setToken(data.token);
        },
    });

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    if (!site) return null;

    const tabs = [
        { id: 'overview', label: 'Overview', icon: LayoutDashboardIcon },
        { id: 'hosts', label: 'Hosts', icon: Globe },
        { id: 'vms', label: 'VMs', icon: Box },
        { id: 'enrollment', label: 'Enrollment', icon: ShieldCheck },
    ];

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <div className="rounded-xl bg-indigo-600 p-3 text-white">
                        <Globe className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900">{site.name}</h1>
                        <div className="flex items-center gap-2 text-sm text-slate-500">
                            <span className="font-mono">{site.id}</span>
                            <span>•</span>
                            <span className={cn(
                                "flex items-center gap-1",
                                site.connectivity_state === 'ONLINE' ? "text-green-600" : "text-slate-400"
                            )}>
                                <span className="h-2 w-2 rounded-full bg-current" />
                                {site.connectivity_state}
                            </span>
                        </div>
                    </div>
                </div>
                <div className="flex gap-2">
                    <button className="flex items-center gap-2 rounded-lg border bg-white px-4 py-2 text-sm font-semibold text-slate-700 shadow-sm hover:bg-slate-50">
                        Edit
                    </button>
                    <button
                        onClick={() => setIsWizardOpen(true)}
                        className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
                    >
                        <Plus className="h-4 w-4" />
                        Create VM
                    </button>
                </div>
            </div>

            <div className="border-b border-slate-200">
                <nav className="-mb-px flex gap-8">
                    {tabs.map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={cn(
                                "flex items-center gap-2 border-b-2 py-4 text-sm font-medium transition-colors",
                                activeTab === tab.id
                                    ? "border-indigo-600 text-indigo-600"
                                    : "border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700"
                            )}
                        >
                            <tab.icon className="h-4 w-4" />
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>

            <div className="py-4">
                {activeTab === 'overview' && (
                    <div className="grid gap-6 md:grid-cols-3">
                        <div className="rounded-xl border bg-white p-6 shadow-sm">
                            <div className="flex items-center gap-3">
                                <div className="rounded-lg bg-indigo-50 p-2 text-indigo-600">
                                    <Activity className="h-5 w-5" />
                                </div>
                                <span className="text-sm font-medium text-slate-500">Nodes</span>
                            </div>
                            <p className="mt-4 text-3xl font-bold text-slate-900">{hostsData?.hosts.length || 0}</p>
                        </div>
                        <div className="rounded-xl border bg-white p-6 shadow-sm">
                            <div className="flex items-center gap-3">
                                <div className="rounded-lg bg-green-50 p-2 text-green-600">
                                    <Zap className="h-5 w-5" />
                                </div>
                                <span className="text-sm font-medium text-slate-500">Active VMs</span>
                            </div>
                            <p className="mt-4 text-3xl font-bold text-slate-900">{vmsData?.vms.filter(v => v.state === 'running').length || 0}</p>
                        </div>
                        <div className="rounded-xl border bg-white p-6 shadow-sm">
                            <div className="flex items-center gap-3">
                                <div className="rounded-lg bg-blue-50 p-2 text-blue-600">
                                    <Cpu className="h-5 w-5" />
                                </div>
                                <span className="text-sm font-medium text-slate-500">Total vCPU</span>
                            </div>
                            <p className="mt-4 text-3xl font-bold text-slate-900">{vmsData?.vms.reduce((acc, v) => acc + v.vcpu_count, 0) || 0}</p>
                        </div>
                    </div>
                )}

                {activeTab === 'hosts' && (
                    <div className="rounded-xl border bg-white shadow-sm overflow-hidden">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-slate-50 border-b text-slate-500 font-medium">
                                <tr>
                                    <th className="px-6 py-3">Host</th>
                                    <th className="px-6 py-3">OS / Arch</th>
                                    <th className="px-6 py-3">Resources</th>
                                    <th className="px-6 py-3">Status</th>
                                    <th className="px-6 py-3">Updated</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {(hostsData?.hosts || []).map((host) => (
                                    <tr key={host.id}>
                                        <td className="px-6 py-4">
                                            <div className="font-semibold text-slate-900">{host.hostname}</div>
                                            <div className="text-[10px] text-slate-400 font-mono">{host.id}</div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="flex items-center gap-1">
                                                <span className="capitalize">{host.last_facts?.os_type || '-'}</span>
                                                <span className="text-slate-300">/</span>
                                                <span>{host.last_facts?.arch || '-'}</span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <div className="space-y-1">
                                                <div className="flex items-center gap-2 text-xs">
                                                    <Cpu className="h-3 w-3 text-slate-400" />
                                                    {host.last_facts?.cpu_cores || 0} Cores
                                                </div>
                                                <div className="flex items-center gap-2 text-xs">
                                                    <HardDrive className="h-3 w-3 text-slate-400" />
                                                    {formatBytes(host.last_facts?.memory_total || 0)} RAM
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className="inline-flex items-center gap-1.5 rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">
                                                <span className="h-1.5 w-1.5 rounded-full bg-green-600" />
                                                ONLINE
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-slate-500">
                                            {relativeTime(host.updated_at)}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}

                {activeTab === 'vms' && (
                    <div className="rounded-xl border bg-white shadow-sm overflow-hidden">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-slate-50 border-b text-slate-500 font-medium">
                                <tr>
                                    <th className="px-6 py-3">MicroVM</th>
                                    <th className="px-6 py-3">Host</th>
                                    <th className="px-6 py-3">Resources</th>
                                    <th className="px-6 py-3">State</th>
                                    <th className="px-6 py-3">Transition</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y">
                                {(vmsData?.vms || []).map((vm) => (
                                    <tr key={vm.id} className="hover:bg-slate-50 transition-colors">
                                        <td className="px-6 py-4">
                                            <div className="font-semibold text-slate-900">{vm.name}</div>
                                            <div className="text-[10px] text-slate-400 font-mono">{vm.id}</div>
                                        </td>
                                        <td className="px-6 py-4 text-slate-500">
                                            {vm.host_id || 'unassigned'}
                                        </td>
                                        <td className="px-6 py-4 text-slate-500">
                                            {vm.vcpu_count} vCPU / {vm.memory_mib} MiB
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className={cn(
                                                "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium uppercase tracking-wider",
                                                vm.state === 'running' ? "bg-green-50 text-green-700" : "bg-slate-100 text-slate-600"
                                            )}>
                                                {vm.state}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 text-slate-500">
                                            {relativeTime(vm.last_transition_at || '')}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}

                {activeTab === 'enrollment' && (
                    <div className="max-w-2xl space-y-6">
                        <div className="rounded-xl border bg-white p-6 shadow-sm">
                            <h3 className="text-lg font-semibold text-slate-900">Enrollment Token</h3>
                            <p className="mt-2 text-sm text-slate-500">Generate a one-time token to onboard a new edge agent to this site.</p>

                            {!token ? (
                                <button
                                    onClick={() => createTokenMutation.mutate()}
                                    disabled={createTokenMutation.isPending}
                                    className="mt-6 flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50"
                                >
                                    {createTokenMutation.isPending ? <Loader2Icon className="h-4 w-4 animate-spin" /> : <ShieldCheck className="h-4 w-4" />}
                                    Generate One-Time Token
                                </button>
                            ) : (
                                <div className="mt-6 space-y-4">
                                    <div className="rounded-lg bg-slate-900 p-4 font-mono text-sm text-white flex items-center justify-between">
                                        <span>{token}</span>
                                        <button
                                            onClick={() => copyToClipboard(token)}
                                            className="text-slate-400 hover:text-white transition-colors"
                                        >
                                            {copied ? <Check className="h-4 w-4 text-green-400" /> : <Copy className="h-4 w-4" />}
                                        </button>
                                    </div>
                                    <div className="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-xs text-amber-800 border border-amber-200">
                                        <AlertCircleIcon className="h-4 w-4 shrink-0" />
                                        <p>This token will only be shown once. Copy it now and use it within 24 hours.</p>
                                    </div>
                                </div>
                            )}
                        </div>

                        <div className="rounded-xl border bg-white p-6 shadow-sm">
                            <h3 className="text-lg font-semibold text-slate-900">Install Command</h3>
                            <p className="mt-2 text-sm text-slate-500">Run this command on your edge host to install and enroll the agent.</p>

                            <div className="mt-6 space-y-2">
                                <div className="rounded-lg bg-slate-900 p-4 font-mono text-xs text-indigo-300 overflow-x-auto">
                                    <div className="flex justify-between items-center group">
                                        <code>sudo nkudo-edge enroll --control-plane {authStore.getState().config?.baseUrl} --token {token || '<TOKEN>'}</code>
                                        <button onClick={() => copyToClipboard(`sudo nkudo-edge enroll --control-plane ${authStore.getState().config?.baseUrl} --token ${token || '<TOKEN>'}`)} className="opacity-0 group-hover:opacity-100 transition-opacity">
                                            <Copy className="h-4 w-4" />
                                        </button>
                                    </div>
                                    <div className="mt-2 text-slate-500"># Then start the service</div>
                                    <div className="flex justify-between items-center group">
                                        <code>sudo systemctl enable --now nkudo-edge</code>
                                        <button onClick={() => copyToClipboard(`sudo systemctl enable --now nkudo-edge`)} className="opacity-0 group-hover:opacity-100 transition-opacity">
                                            <Copy className="h-4 w-4" />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )}
            </div>

            {isWizardOpen && (
                <CreateVMWizard
                    siteId={siteId}
                    onClose={() => setIsWizardOpen(false)}
                />
            )}
        </div>
    );
}
