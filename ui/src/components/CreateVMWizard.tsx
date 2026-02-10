'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import {
    X,
    Cpu,
    HardDrive,
    Zap,
    Network,
    Layers,
    Loader2,
    Box,
    Plus
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface CreateVMWizardProps {
    siteId: string;
    onClose: () => void;
}

const templates = [
    { id: 'ubuntu-ssh', name: 'Ubuntu 22.04 LTS (SSH)', icon: Box, description: 'Standard Ubuntu server with SSH pre-configured' },
    { id: 'k3s-node', name: 'K3s Worker Node', icon: Layers, description: 'Lightweight Kubernetes node ready to join a cluster' },
    { id: 'minimal-alpine', name: 'Alpine Linux', icon: Zap, description: 'Minimal footprint for edge microservices' },
];

export function CreateVMWizard({ siteId, onClose }: CreateVMWizardProps) {
    const [name, setName] = useState('');
    const [vcpu, setVcpu] = useState(1);
    const [ram, setRam] = useState(512);
    const [disk, setDisk] = useState(10);
    const [template, setTemplate] = useState('ubuntu-ssh');
    const router = useRouter();
    const client = authStore.getState().client;

    const mutation = useMutation({
        mutationFn: async () => {
            const actions = [
                {
                    operation: 'CREATE',
                    name: name,
                    vcpu_count: vcpu,
                    memory_mib: ram,
                    rootfs_path: `/images/${template}.img`, // Mock path for demo
                },
                {
                    operation: 'START',
                }
            ];
            return client!.applyPlan(siteId, actions);
        },
        onSuccess: (data) => {
            const executionId = data.executions[0]?.id;
            if (executionId) {
                router.push(`/executions/${executionId}`);
            } else {
                router.push(`/sites/${siteId}`);
            }
        }
    });

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm">
            <div className="w-full max-w-2xl rounded-2xl bg-white shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
                <div className="flex items-center justify-between border-b px-6 py-4">
                    <h2 className="text-xl font-bold text-slate-900">Create MicroVM</h2>
                    <button onClick={onClose} className="rounded-full p-1 hover:bg-slate-100 transition-colors">
                        <X className="h-5 w-5 text-slate-500" />
                    </button>
                </div>

                <div className="flex-1 overflow-y-auto p-6 space-y-8">
                    <section className="space-y-4">
                        <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Basic Info</h3>
                        <div className="grid gap-4 sm:grid-cols-1">
                            <div>
                                <label className="block text-sm font-medium text-slate-700">VM Display Name</label>
                                <input
                                    type="text"
                                    required
                                    placeholder="e.g. edge-db-01"
                                    value={name}
                                    onChange={(e) => setName(e.target.value)}
                                    className="mt-1 block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                                />
                            </div>
                        </div>
                    </section>

                    <section className="space-y-4">
                        <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Template</h3>
                        <div className="grid gap-3 sm:grid-cols-1">
                            {templates.map((t) => (
                                <button
                                    key={t.id}
                                    onClick={() => setTemplate(t.id)}
                                    className={cn(
                                        "flex items-start gap-4 rounded-xl border p-4 text-left transition-all",
                                        template === t.id
                                            ? "border-indigo-600 bg-indigo-50/50 ring-1 ring-indigo-600"
                                            : "border-slate-200 hover:border-indigo-300"
                                    )}
                                >
                                    <div className={cn(
                                        "rounded-lg p-2 mt-1",
                                        template === t.id ? "bg-indigo-600 text-white" : "bg-slate-100 text-slate-500"
                                    )}>
                                        <t.icon className="h-5 w-5" />
                                    </div>
                                    <div>
                                        <p className="font-semibold text-slate-900">{t.name}</p>
                                        <p className="text-sm text-slate-500">{t.description}</p>
                                    </div>
                                </button>
                            ))}
                        </div>
                    </section>

                    <section className="space-y-4">
                        <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Resources</h3>
                        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
                            <div className="space-y-2">
                                <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
                                    <Cpu className="h-4 w-4 text-slate-400" />
                                    vCPU Cores
                                </label>
                                <div className="flex items-center gap-3">
                                    <input
                                        type="range" min="1" max="8" step="1"
                                        value={vcpu} onChange={(e) => setVcpu(parseInt(e.target.value))}
                                        className="w-full"
                                    />
                                    <span className="w-8 text-center font-bold">{vcpu}</span>
                                </div>
                            </div>
                            <div className="space-y-2">
                                <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
                                    <Box className="h-4 w-4 text-slate-400" />
                                    Memory (MB)
                                </label>
                                <select
                                    value={ram} onChange={(e) => setRam(parseInt(e.target.value))}
                                    className="block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500"
                                >
                                    <option value={128}>128 MB</option>
                                    <option value={256}>256 MB</option>
                                    <option value={512}>512 MB</option>
                                    <option value={1024}>1024 MB</option>
                                    <option value={2048}>2048 MB</option>
                                </select>
                            </div>
                            <div className="space-y-2">
                                <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
                                    <HardDrive className="h-4 w-4 text-slate-400" />
                                    Disk (GB)
                                </label>
                                <input
                                    type="number"
                                    min="1" max="100"
                                    value={disk}
                                    onChange={(e) => setDisk(parseInt(e.target.value))}
                                    className="block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500"
                                />
                            </div>
                        </div>
                    </section>

                    <section className="space-y-4">
                        <h3 className="text-sm font-bold uppercase tracking-wider text-slate-400">Network</h3>
                        <div className="flex items-center gap-3 rounded-xl border border-slate-200 p-4 bg-slate-50">
                            <Network className="h-5 w-5 text-slate-400" />
                            <div>
                                <p className="text-sm font-medium text-slate-900">Virtual Bridge: br0</p>
                                <p className="text-xs text-slate-500">Standard Linux bridge for host networking</p>
                            </div>
                        </div>
                    </section>
                </div>

                <div className="border-t bg-slate-50 px-6 py-4 flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        className="rounded-lg px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-200 transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={() => mutation.mutate()}
                        disabled={!name || mutation.isPending}
                        className="flex items-center gap-2 rounded-lg bg-indigo-600 px-6 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50"
                    >
                        {mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                        Deploy VM
                    </button>
                </div>
            </div>
        </div>
    );
}
