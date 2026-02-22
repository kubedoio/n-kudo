'use client';

import { useState, useRef } from 'react';
import { useDesignerStore } from '@/store/designer';
import { cn } from '@/lib/utils';
import {
  Save,
  FolderOpen,
  Trash2,
  Rocket,
  Plus,
  AlertCircle,
  Check,
  X,
  FileJson,
  Upload,
} from 'lucide-react';
import type { Template } from '@/types/designer';

interface ToolbarProps {
  onSave?: (template: Template) => void;
  onLoad?: (template: Template) => void;
  onDeploy?: () => void;
  onClear?: () => void;
}

export function Toolbar({ onSave, onLoad, onDeploy, onClear }: ToolbarProps) {
  const store = useDesignerStore();
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false);
  const [isLoadDialogOpen, setIsLoadDialogOpen] = useState(false);
  const [isClearConfirmOpen, setIsClearConfirmOpen] = useState(false);
  const [templateName, setTemplateName] = useState('');
  const [templateDescription, setTemplateDescription] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Get current template name if editing
  const currentName = store.currentTemplate?.name || '';
  const isDirty = store.isDirty;

  // Handle save
  const handleSave = () => {
    if (!templateName.trim()) return;

    const template: Template = {
      id: store.currentTemplate?.id || `template_${Date.now()}`,
      name: templateName,
      description: templateDescription,
      nodes: store.nodes as any,
      edges: store.edges as any,
      createdAt: store.currentTemplate?.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    onSave?.(template);
    store.setCurrentTemplate(template);
    setIsSaveDialogOpen(false);
    setTemplateName('');
    setTemplateDescription('');
  };

  // Handle load from file
  const handleFileLoad = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const template = JSON.parse(e.target?.result as string) as Template;
        onLoad?.(template);
        store.loadTemplate(template);
        setIsLoadDialogOpen(false);
      } catch (err) {
        alert('Invalid template file');
      }
    };
    reader.readAsText(file);

    // Reset input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  // Handle clear
  const handleClear = () => {
    onClear?.();
    store.clearCanvas();
    setIsClearConfirmOpen(false);
  };

  // Handle deploy
  const handleDeploy = () => {
    store.setDeploying(true);
    onDeploy?.();
    // Reset after a delay (in real app, this would wait for deployment)
    setTimeout(() => store.setDeploying(false), 1000);
  };

  // Open save dialog with current name
  const openSaveDialog = () => {
    setTemplateName(currentName || 'New Template');
    setTemplateDescription(store.currentTemplate?.description || '');
    setIsSaveDialogOpen(true);
  };

  return (
    <>
      {/* Main Toolbar */}
      <div className="flex items-center justify-between border-b bg-white px-4 py-2">
        {/* Left: Template Info */}
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            {currentName ? (
              <>
                <span className="text-sm font-semibold text-slate-900">
                  {currentName}
                </span>
                {isDirty && (
                  <span className="text-xs text-amber-600">(unsaved)</span>
                )}
              </>
            ) : (
              <span className="text-sm italic text-slate-400">
                Untitled Template
              </span>
            )}
          </div>
        </div>

        {/* Center: Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={openSaveDialog}
            className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100"
          >
            <Save className="h-4 w-4" />
            Save
          </button>

          <button
            onClick={() => setIsLoadDialogOpen(true)}
            className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100"
          >
            <FolderOpen className="h-4 w-4" />
            Load
          </button>

          <div className="mx-2 h-5 w-px bg-slate-200" />

          <button
            onClick={() => setIsClearConfirmOpen(true)}
            disabled={store.nodes.length === 0}
            className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50 disabled:hover:bg-transparent"
          >
            <Trash2 className="h-4 w-4" />
            Clear
          </button>
        </div>

        {/* Right: Deploy */}
        <button
          onClick={handleDeploy}
          disabled={store.nodes.length === 0 || store.isDeploying}
          className={cn(
            'flex items-center gap-2 rounded-md px-4 py-1.5 text-sm font-semibold text-white transition-all',
            store.nodes.length === 0
              ? 'cursor-not-allowed bg-slate-300'
              : 'bg-indigo-600 hover:bg-indigo-700 active:scale-95'
          )}
        >
          {store.isDeploying ? (
            <>
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              Deploying...
            </>
          ) : (
            <>
              <Rocket className="h-4 w-4" />
              Deploy to Site
            </>
          )}
        </button>
      </div>

      {/* Save Dialog */}
      {isSaveDialogOpen && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-96 rounded-lg bg-white p-6 shadow-xl">
            <h3 className="mb-4 text-lg font-semibold text-slate-900">
              Save Template
            </h3>

            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-slate-700">
                  Template Name
                </label>
                <input
                  type="text"
                  value={templateName}
                  onChange={(e) => setTemplateName(e.target.value)}
                  placeholder="My Infrastructure Template"
                  className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  autoFocus
                />
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-slate-700">
                  Description (optional)
                </label>
                <textarea
                  value={templateDescription}
                  onChange={(e) => setTemplateDescription(e.target.value)}
                  placeholder="Brief description of this template..."
                  rows={3}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setIsSaveDialogOpen(false)}
                className="rounded-md px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={!templateName.trim()}
                className="flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:opacity-50"
              >
                <Check className="h-4 w-4" />
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Load Dialog */}
      {isLoadDialogOpen && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-96 rounded-lg bg-white p-6 shadow-xl">
            <h3 className="mb-4 text-lg font-semibold text-slate-900">
              Load Template
            </h3>

            <div className="space-y-3">
              <p className="text-sm text-slate-600">
                Select a template JSON file to load:
              </p>

              <label className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-slate-300 p-6 transition-colors hover:border-indigo-400 hover:bg-indigo-50">
                <Upload className="mb-2 h-8 w-8 text-slate-400" />
                <span className="text-sm font-medium text-slate-600">
                  Click to upload
                </span>
                <span className="text-xs text-slate-400">or drag and drop</span>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".json"
                  onChange={handleFileLoad}
                  className="hidden"
                />
              </label>

              <div className="flex items-center gap-2 text-xs text-slate-500">
                <FileJson className="h-4 w-4" />
                <span>Only .json files are supported</span>
              </div>
            </div>

            <div className="mt-6 flex justify-end">
              <button
                onClick={() => setIsLoadDialogOpen(false)}
                className="rounded-md px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Clear Confirmation Dialog */}
      {isClearConfirmOpen && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-96 rounded-lg bg-white p-6 shadow-xl">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100">
                <AlertCircle className="h-5 w-5 text-red-600" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-slate-900">
                  Clear Canvas?
                </h3>
                <p className="text-sm text-slate-500">
                  This will remove all nodes and edges.
                </p>
              </div>
            </div>

            {isDirty && (
              <div className="mb-4 rounded-lg bg-amber-50 p-3 text-sm text-amber-700">
                <strong>Warning:</strong> You have unsaved changes that will be
                lost.
              </div>
            )}

            <div className="flex justify-end gap-3">
              <button
                onClick={() => setIsClearConfirmOpen(false)}
                className="rounded-md px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100"
              >
                Cancel
              </button>
              <button
                onClick={handleClear}
                className="flex items-center gap-2 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700"
              >
                <Trash2 className="h-4 w-4" />
                Clear Canvas
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
