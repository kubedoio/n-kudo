'use client';

import { useState, useCallback, useEffect } from 'react';
import {
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  ReactFlowProvider,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useDesignerStore } from './store';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import {
  Canvas,
  Palette,
  PropertyPanel,
  EmptyState,
  TemplateBrowser,
  SaveTemplateModal,
  DeployModal,
} from './components';
import { createDefaultNodeData, generateId } from './utils';
import {
  Plus,
  Save,
  FolderOpen,
  Rocket,
  Trash2,
  Undo,
  Sparkles,
} from 'lucide-react';
import { cn } from '@/lib/utils';
// Types imported from global types
import type { DesignerNode, DesignerEdge } from '@/types/designer';

// Wrap the main content with ReactFlowProvider for proper context
function DesignerContent() {
  const store = useDesignerStore();
  const [nodes, setNodes, onNodesChange] = useNodesState<DesignerNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<DesignerEdge>([]);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [showTemplateBrowser, setShowTemplateBrowser] = useState(false);
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [notification, setNotification] = useState<{
    message: string;
    type: 'success' | 'error';
  } | null>(null);

  // Sync store state with React Flow state
  useEffect(() => {
    setNodes(store.nodes);
  }, [store.nodes, setNodes]);

  useEffect(() => {
    setEdges(store.edges);
  }, [store.edges, setEdges]);

  // Show notification helper
  const showNotification = useCallback(
    (message: string, type: 'success' | 'error' = 'success') => {
      setNotification({ message, type });
      setTimeout(() => setNotification(null), 3000);
    },
    []
  );

  // Connection handler
  const onConnect = useCallback(
    (params: Connection) => {
      if (!params.source || !params.target) return;
      
      const newEdge: DesignerEdge = {
        id: `e-${params.source}-${params.target}`,
        source: params.source,
        target: params.target,
        sourceHandle: params.sourceHandle || undefined,
        targetHandle: params.targetHandle || undefined,
      };
      
      setEdges((eds) => addEdge(params, eds));
      store.setEdges([...store.edges, newEdge]);
    },
    [setEdges, store]
  );

  // Node click handler
  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: DesignerNode) => {
      setSelectedNodeId(node.id);
      store.selectNode(node.id);
    },
    [store]
  );

  // Pane click handler (deselect)
  const onPaneClick = useCallback(() => {
    setSelectedNodeId(null);
    store.selectNode(null);
  }, [store]);

  // Add node handler
  const handleAddNode = useCallback(
    (type: 'vm' | 'network' | 'volume') => {
      const position = {
        x: 100 + Math.random() * 200,
        y: 100 + Math.random() * 200,
      };
      store.addNode(type, position);
      showNotification(`${type.toUpperCase()} node added`, 'success');
    },
    [store, showNotification]
  );

  // Delete selected node
  const handleDeleteSelected = useCallback(() => {
    if (selectedNodeId) {
      store.removeNode(selectedNodeId);
      setSelectedNodeId(null);
      showNotification('Node deleted', 'success');
    }
  }, [selectedNodeId, store, showNotification]);

  // Save template handler
  const handleSaveTemplate = useCallback(
    (template: { name: string; description: string }) => {
      showNotification(`Template "${template.name}" saved successfully!`, 'success');
    },
    [showNotification]
  );

  // Load template handler
  const handleLoadTemplate = useCallback(
    (template: { nodes: DesignerNode[]; edges: DesignerEdge[] }) => {
      store.setNodes(template.nodes);
      store.setEdges(template.edges);
      setSelectedNodeId(null);
      showNotification('Template loaded successfully!', 'success');
    },
    [store, showNotification]
  );

  // Clear canvas handler
  const handleClear = useCallback(() => {
    store.clearCanvas();
    setSelectedNodeId(null);
    showNotification('Canvas cleared', 'success');
  }, [store, showNotification]);

  // Keyboard shortcuts
  useKeyboardShortcuts({
    onDelete: handleDeleteSelected,
    onSave: () => setShowSaveModal(true),
  });

  const hasNodes = nodes.length > 0;

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-200 bg-white px-6 py-3">
        <div className="flex items-center gap-3">
          <div>
            <h1 className="text-lg font-bold text-slate-900">
              Infrastructure Designer
            </h1>
            <p className="text-xs text-slate-500">
              Design and deploy infrastructure templates
            </p>
          </div>
          <span className="ml-2 rounded-full bg-indigo-50 px-2 py-0.5 text-[10px] font-medium text-indigo-700">
            MVP-2
          </span>
        </div>

        {/* Toolbar Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowTemplateBrowser(true)}
            className={cn(
              'flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              'border border-slate-200 text-slate-700 hover:bg-slate-50 hover:text-indigo-600'
            )}
          >
            <FolderOpen className="h-4 w-4" />
            Load
          </button>
          <button
            onClick={() => setShowSaveModal(true)}
            disabled={!hasNodes}
            className={cn(
              'flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              hasNodes
                ? 'border border-slate-200 text-slate-700 hover:bg-slate-50 hover:text-indigo-600'
                : 'cursor-not-allowed border border-slate-100 text-slate-400'
            )}
          >
            <Save className="h-4 w-4" />
            Save
          </button>
          <button
            onClick={handleDeleteSelected}
            disabled={!selectedNodeId}
            className={cn(
              'flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              selectedNodeId
                ? 'border border-slate-200 text-slate-700 hover:bg-red-50 hover:text-red-600'
                : 'cursor-not-allowed border border-slate-100 text-slate-400'
            )}
          >
            <Trash2 className="h-4 w-4" />
            Delete
          </button>
          <button
            onClick={handleClear}
            disabled={!hasNodes}
            className={cn(
              'flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              hasNodes
                ? 'border border-slate-200 text-slate-700 hover:bg-slate-50 hover:text-slate-600'
                : 'cursor-not-allowed border border-slate-100 text-slate-400'
            )}
          >
            <Undo className="h-4 w-4" />
            Clear
          </button>
          <div className="mx-2 h-6 w-px bg-slate-200" />
          <button
            onClick={() => setShowDeployModal(true)}
            disabled={!hasNodes}
            className={cn(
              'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors',
              hasNodes
                ? 'bg-indigo-600 hover:bg-indigo-700'
                : 'cursor-not-allowed bg-slate-300'
            )}
          >
            <Rocket className="h-4 w-4" />
            Deploy
          </button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Node Palette */}
        <Palette />

        {/* Center: Canvas */}
        <div className="relative flex-1" onClick={onPaneClick}>
          {hasNodes ? (
            <Canvas
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onNodeClick={onNodeClick}
            />
          ) : (
            <EmptyState onAddNode={handleAddNode} />
          )}
        </div>

        {/* Right: Property Panel */}
        <PropertyPanel />
      </div>

      {/* Modals */}
      {showTemplateBrowser && (
        <TemplateBrowser
          isOpen={showTemplateBrowser}
          onClose={() => setShowTemplateBrowser(false)}
          onLoad={handleLoadTemplate}
          onNew={() => {
            handleClear();
            setShowTemplateBrowser(false);
          }}
        />
      )}
      {showSaveModal && (
        <SaveTemplateModal
          nodes={nodes as any[]}
          edges={edges as any[]}
          onClose={() => setShowSaveModal(false)}
          onSave={handleSaveTemplate}
        />
      )}
      {showDeployModal && (
        <DeployModal
          nodes={nodes as any[]}
          edges={edges as any[]}
          onClose={() => setShowDeployModal(false)}
        />
      )}

      {/* Notification Toast */}
      {notification && (
        <div
          className={cn(
            'fixed bottom-4 right-4 z-50 rounded-lg px-4 py-3 shadow-lg transition-all',
            notification.type === 'success'
              ? 'bg-emerald-500 text-white'
              : 'bg-red-500 text-white'
          )}
        >
          {notification.message}
        </div>
      )}
    </div>
  );
}

// Main page component with ReactFlowProvider
export default function DesignerPage() {
  return (
    <ReactFlowProvider>
      <DesignerContent />
    </ReactFlowProvider>
  );
}
