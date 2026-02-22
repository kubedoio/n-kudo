'use client';

import { useState } from 'react';
import {
    FolderOpen,
    Plus,
    Trash2,
    X,
    FileJson,
    Clock,
    AlertCircle,
    Search,
    Loader2,
} from 'lucide-react';
import { useTemplates, useDeleteTemplate } from '../hooks';
import { useDesignerStore } from '@/store/designer';
import type { Template } from '@/api/types';
import { formatDistanceToNow } from 'date-fns';

interface TemplateBrowserProps {
    isOpen: boolean;
    onClose: () => void;
    onLoad: (template: Template) => void;
    onNew: () => void;
}

export function TemplateBrowser({ isOpen, onClose, onLoad, onNew }: TemplateBrowserProps) {
    const [searchQuery, setSearchQuery] = useState('');
    const [templateToDelete, setTemplateToDelete] = useState<Template | null>(null);

    const { data, isLoading, error } = useTemplates();
    const deleteMutation = useDeleteTemplate();
    const store = useDesignerStore();

    if (!isOpen) return null;

    const templates = data?.templates || [];

    // Filter templates by search query
    const filteredTemplates = templates.filter((template) =>
        template.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        template.description?.toLowerCase().includes(searchQuery.toLowerCase())
    );

    // Sort by updated_at descending (most recent first)
    const sortedTemplates = [...filteredTemplates].sort(
        (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    );

    const handleLoad = (template: Template) => {
        onLoad(template);
        onClose();
    };

    const handleDelete = async (template: Template) => {
        if (!templateToDelete) {
            setTemplateToDelete(template);
            return;
        }

        try {
            await deleteMutation.mutateAsync(template.id);
            setTemplateToDelete(null);
        } catch (err) {
            // Error is handled by the mutation
        }
    };

    const handleNew = () => {
        store.clearCanvas();
        onNew();
        onClose();
    };

    const formatDate = (dateString: string) => {
        try {
            return formatDistanceToNow(new Date(dateString), { addSuffix: true });
        } catch {
            return 'Unknown date';
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
            <div className="flex h-[600px] w-[800px] flex-col rounded-lg bg-white shadow-xl">
                {/* Header */}
                <div className="flex items-center justify-between border-b px-6 py-4">
                    <div className="flex items-center gap-3">
                        <FolderOpen className="h-5 w-5 text-indigo-600" />
                        <h2 className="text-lg font-semibold text-slate-900">
                            Load Template
                        </h2>
                    </div>
                    <button
                        onClick={onClose}
                        className="rounded-md p-1 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                    >
                        <X className="h-5 w-5" />
                    </button>
                </div>

                {/* Search and New Button */}
                <div className="flex items-center gap-4 border-b px-6 py-3">
                    <div className="relative flex-1">
                        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                        <input
                            type="text"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            placeholder="Search templates..."
                            className="w-full rounded-md border border-slate-300 py-2 pl-9 pr-4 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        />
                    </div>
                    <button
                        onClick={handleNew}
                        className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                    >
                        <Plus className="h-4 w-4" />
                        New Template
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-auto p-6">
                    {isLoading ? (
                        <div className="flex h-full flex-col items-center justify-center gap-3">
                            <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
                            <p className="text-sm text-slate-500">Loading templates...</p>
                        </div>
                    ) : error ? (
                        <div className="flex h-full flex-col items-center justify-center gap-3">
                            <AlertCircle className="h-12 w-12 text-red-400" />
                            <p className="text-sm text-slate-600">Failed to load templates</p>
                            <p className="text-xs text-slate-400">
                                {error instanceof Error ? error.message : 'Unknown error'}
                            </p>
                        </div>
                    ) : sortedTemplates.length === 0 ? (
                        <div className="flex h-full flex-col items-center justify-center gap-4">
                            <FileJson className="h-16 w-16 text-slate-200" />
                            <div className="text-center">
                                <p className="text-sm font-medium text-slate-600">
                                    {searchQuery ? 'No templates match your search' : 'No templates yet'}
                                </p>
                                <p className="mt-1 text-xs text-slate-400">
                                    {searchQuery
                                        ? 'Try adjusting your search terms'
                                        : 'Create your first template to get started'}
                                </p>
                            </div>
                            {!searchQuery && (
                                <button
                                    onClick={handleNew}
                                    className="flex items-center gap-2 rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50"
                                >
                                    <Plus className="h-4 w-4" />
                                    Create New Template
                                </button>
                            )}
                        </div>
                    ) : (
                        <div className="space-y-2">
                            {sortedTemplates.map((template) => (
                                <div
                                    key={template.id}
                                    className="group flex items-center justify-between rounded-lg border border-slate-200 bg-white p-4 transition-all hover:border-indigo-300 hover:shadow-sm"
                                >
                                    <div className="min-w-0 flex-1">
                                        <h3 className="truncate text-sm font-medium text-slate-900">
                                            {template.name}
                                        </h3>
                                        {template.description && (
                                            <p className="mt-1 truncate text-xs text-slate-500">
                                                {template.description}
                                            </p>
                                        )}
                                        <div className="mt-2 flex items-center gap-4 text-xs text-slate-400">
                                            <span className="flex items-center gap-1">
                                                <Clock className="h-3 w-3" />
                                                Updated {formatDate(template.updated_at)}
                                            </span>
                                            <span>
                                                {template.nodes?.length || 0} nodes
                                            </span>
                                        </div>
                                    </div>

                                    <div className="ml-4 flex items-center gap-2">
                                        {templateToDelete?.id === template.id ? (
                                            <>
                                                <span className="text-xs text-slate-500">
                                                    Confirm?
                                                </span>
                                                <button
                                                    onClick={() => handleDelete(template)}
                                                    disabled={deleteMutation.isPending}
                                                    className="flex h-8 w-8 items-center justify-center rounded-md bg-red-100 text-red-600 transition-colors hover:bg-red-200"
                                                >
                                                    {deleteMutation.isPending ? (
                                                        <Loader2 className="h-4 w-4 animate-spin" />
                                                    ) : (
                                                        <Trash2 className="h-4 w-4" />
                                                    )}
                                                </button>
                                                <button
                                                    onClick={() => setTemplateToDelete(null)}
                                                    disabled={deleteMutation.isPending}
                                                    className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-100 text-slate-600 transition-colors hover:bg-slate-200"
                                                >
                                                    <X className="h-4 w-4" />
                                                </button>
                                            </>
                                        ) : (
                                            <>
                                                <button
                                                    onClick={() => setTemplateToDelete(template)}
                                                    className="flex h-8 w-8 items-center justify-center rounded-md text-slate-400 opacity-0 transition-all hover:bg-red-50 hover:text-red-600 group-hover:opacity-100"
                                                    title="Delete template"
                                                >
                                                    <Trash2 className="h-4 w-4" />
                                                </button>
                                                <button
                                                    onClick={() => handleLoad(template)}
                                                    className="rounded-md bg-indigo-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                                                >
                                                    Load
                                                </button>
                                            </>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="border-t px-6 py-4">
                    <p className="text-xs text-slate-400">
                        {templates.length} template{templates.length !== 1 ? 's' : ''} available
                    </p>
                </div>
            </div>
        </div>
    );
}
