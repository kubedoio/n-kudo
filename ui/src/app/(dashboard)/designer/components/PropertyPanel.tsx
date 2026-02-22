'use client';

import { useDesignerStore } from '../store';
import { VMNodeData, NetworkNodeData, VolumeNodeData, NodeType } from '../types';
import { VMPropertyForm } from './VMPropertyForm';
import { NetworkPropertyForm } from './NetworkPropertyForm';
import { VolumePropertyForm } from './VolumePropertyForm';
import { Server, Network, Database, Trash2 } from 'lucide-react';

export function PropertyPanel() {
  const { selectedNodeId, nodes, updateNodeData, removeNode } = useDesignerStore();

  const selectedNode = nodes.find((n) => n.id === selectedNodeId) || null;

  if (!selectedNode) {
    return (
      <div className="w-80 border-l bg-white p-4 flex flex-col h-full">
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-slate-500 text-center">
            Select a component to configure
          </p>
        </div>
      </div>
    );
  }

  const nodeType = selectedNode.data.type;
  const nodeData = selectedNode.data;

  const getNodeIcon = () => {
    switch (nodeType) {
      case 'vm':
        return (
          <div className="flex h-8 w-8 items-center justify-center rounded bg-indigo-100 text-indigo-600">
            <Server className="h-4 w-4" />
          </div>
        );
      case 'network':
        return (
          <div className="flex h-8 w-8 items-center justify-center rounded bg-emerald-100 text-emerald-600">
            <Network className="h-4 w-4" />
          </div>
        );
      case 'volume':
        return (
          <div className="flex h-8 w-8 items-center justify-center rounded bg-amber-100 text-amber-600">
            <Database className="h-4 w-4" />
          </div>
        );
      default:
        return null;
    }
  };

  const getNodeTypeLabel = () => {
    switch (nodeType) {
      case 'vm':
        return 'Virtual Machine';
      case 'network':
        return 'Network';
      case 'volume':
        return 'Volume';
      default:
        return 'Component';
    }
  };

  const handleDelete = () => {
    if (selectedNode) {
      removeNode(selectedNode.id);
    }
  };

  const renderForm = () => {
    switch (nodeType) {
      case 'vm':
        return (
          <VMPropertyForm
            data={nodeData as unknown as VMNodeData}
            onChange={(data) => updateNodeData(selectedNode.id, data)}
          />
        );
      case 'network':
        return (
          <NetworkPropertyForm
            data={nodeData as unknown as NetworkNodeData}
            onChange={(data) => updateNodeData(selectedNode.id, data)}
          />
        );
      case 'volume':
        return (
          <VolumePropertyForm
            data={nodeData as unknown as VolumeNodeData}
            onChange={(data) => updateNodeData(selectedNode.id, data)}
          />
        );
      default:
        return (
          <p className="text-sm text-slate-500">
            Unknown component type
          </p>
        );
    }
  };

  return (
    <div className="w-80 border-l bg-white flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-3 border-b p-4">
        {getNodeIcon()}
        <div className="flex-1 min-w-0">
          <h2 className="text-sm font-semibold text-slate-900 truncate">
            {(nodeData as unknown as { name: string }).name}
          </h2>
          <p className="text-xs text-slate-500">
            {getNodeTypeLabel()}
          </p>
        </div>
      </div>

      {/* Form Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {renderForm()}
      </div>

      {/* Footer with Delete Button */}
      <div className="border-t p-4">
        <button
          onClick={handleDelete}
          className="flex w-full items-center justify-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-100"
        >
          <Trash2 className="h-4 w-4" />
          Delete Node
        </button>
      </div>
    </div>
  );
}
