'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import { X, Globe, Loader2, Plus } from 'lucide-react';

interface CreateSiteModalProps {
    onClose: () => void;
}

export function CreateSiteModal({ onClose }: CreateSiteModalProps) {
    const [name, setName] = useState('');
    const client = authStore.getState().client;
    const queryClient = useQueryClient();

    const mutation = useMutation({
        mutationFn: (siteName: string) => client!.createSite(siteName),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['sites'] });
            onClose();
        },
    });

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm">
            <div className="w-full max-w-md rounded-2xl bg-white shadow-2xl overflow-hidden">
                <div className="flex items-center justify-between border-b px-6 py-4">
                    <h2 className="text-xl font-bold text-slate-900">Add New Site</h2>
                    <button onClick={onClose} className="rounded-full p-1 hover:bg-slate-100 transition-colors">
                        <X className="h-5 w-5 text-slate-500" />
                    </button>
                </div>

                <form
                    onSubmit={(e) => {
                        e.preventDefault();
                        mutation.mutate(name);
                    }}
                    className="p-6 space-y-4"
                >
                    <div>
                        <label className="block text-sm font-medium text-slate-700">Site Name</label>
                        <div className="relative mt-1">
                            <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                                <Globe className="h-5 w-5 text-slate-400" />
                            </div>
                            <input
                                type="text"
                                required
                                autoFocus
                                placeholder="e.g. Frankfurt-DC-01"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                className="block w-full rounded-lg border border-slate-300 py-2 pl-10 pr-3 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                            />
                        </div>
                    </div>

                    <div className="flex justify-end gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="rounded-lg px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-100 transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={!name || mutation.isPending}
                            className="flex items-center gap-2 rounded-lg bg-indigo-600 px-6 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50"
                        >
                            {mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                            Create Site
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
