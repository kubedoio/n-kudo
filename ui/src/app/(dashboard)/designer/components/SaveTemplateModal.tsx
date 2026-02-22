'use client';

import { useState } from 'react';
import { X, Save, FileJson, AlertCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { TextInput } from './forms/TextInput';
import type { DesignerNode, DesignerEdge } from '../types';

interface SaveTemplateModalProps {
  nodes: DesignerNode[];
  edges: DesignerEdge[];
  onClose: () => void;
  onSave?: (template: { name: string; description: string }) => void;
}

export function SaveTemplateModal({ nodes, edges, onClose, onSave }: SaveTemplateModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    if (!name.trim()) {
      setError('Please enter a template name');
      return;
    }

    if (nodes.length === 0) {
      setError('Cannot save an empty template');
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      const template = {
        id: `template-${Date.now()}`,
        name: name.trim(),
        description: description.trim(),
        version: '1.0.0',
        nodes: nodes.map((n) => ({
          id: n.id,
          type: n.type,
          position: n.position,
          data: n.data,
        })),
        edges: edges.map((e) => ({
          id: e.id,
          source: e.source,
          target: e.target,
          sourceHandle: e.sourceHandle,
          targetHandle: e.targetHandle,
          label: e.label,
        })),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };

      // Save to localStorage
      const existing = localStorage.getItem('designer-templates');
      const templates = existing ? JSON.parse(existing) : [];
      templates.push(template);
      localStorage.setItem('designer-templates', JSON.stringify(templates));

      // Also download as JSON file
      const blob = new Blob([JSON.stringify(template, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${name.trim().replace(/\s+/g, '_').toLowerCase()}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      onSave?.({ name: name.trim(), description: description.trim() });
      onClose();
    } catch (e) {
      setError('Failed to save template');
      console.error('Save error:', e);
    } finally {
      setIsSaving(false);
    }
  };

  const nodeCounts = {
    vm: nodes.filter((n) => n.type === 'vm').length,
    network: nodes.filter((n) => n.type === 'network').length,
    volume: nodes.filter((n) => n.type === 'volume').length,
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-xl bg-white shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-100">
              <Save className="h-5 w-5 text-indigo-600" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Save Template</h2>
              <p className="text-sm text-slate-500">Save your infrastructure design</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Form */}
        <div className="space-y-4 p-6">
          {error && (
            <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600">
              <AlertCircle className="h-4 w-4" />
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-700">
              Template Name <span className="text-red-500">*</span>
            </label>
            <TextInput
              value={name}
              onChange={setName}
              placeholder="e.g., Production Web Cluster"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-700">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe your infrastructure design..."
              rows={3}
              className="block w-full rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>

          {/* Template Summary */}
          <div className="rounded-lg bg-slate-50 p-4">
            <div className="mb-2 flex items-center gap-2 text-sm font-medium text-slate-700">
              <FileJson className="h-4 w-4" />
              Template Summary
            </div>
            <div className="space-y-1 text-xs text-slate-500">
              <div className="flex justify-between">
                <span>Total Nodes:</span>
                <span className="font-medium text-slate-700">{nodes.length}</span>
              </div>
              <div className="flex justify-between">
                <span>VMs:</span>
                <span className="font-medium text-slate-700">{nodeCounts.vm}</span>
              </div>
              <div className="flex justify-between">
                <span>Networks:</span>
                <span className="font-medium text-slate-700">{nodeCounts.network}</span>
              </div>
              <div className="flex justify-between">
                <span>Volumes:</span>
                <span className="font-medium text-slate-700">{nodeCounts.volume}</span>
              </div>
              <div className="flex justify-between">
                <span>Connections:</span>
                <span className="font-medium text-slate-700">{edges.length}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 border-t border-slate-200 px-6 py-4">
          <button
            onClick={onClose}
            className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving || !name.trim() || nodes.length === 0}
            className={cn(
              'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white',
              isSaving || !name.trim() || nodes.length === 0
                ? 'cursor-not-allowed bg-slate-300'
                : 'bg-indigo-600 hover:bg-indigo-700'
            )}
          >
            <Save className="h-4 w-4" />
            {isSaving ? 'Saving...' : 'Save Template'}
          </button>
        </div>
      </div>
    </div>
  );
}
