'use client';

import { useSyncExternalStore } from 'react';
import type {
  DesignerNode,
  DesignerEdge,
  DesignerNodeData,
  DesignerTemplate,
  VMNodeData,
  NetworkNodeData,
  VolumeNodeData,
} from '../types';
import { NodeType } from '../types';

export interface DesignerState {
  // Canvas state
  nodes: DesignerNode[];
  edges: DesignerEdge[];
  selectedNodeId: string | null;

  // Template metadata
  templateName: string;
  templateDescription: string;
}

const initialState: DesignerState = {
  nodes: [],
  edges: [],
  selectedNodeId: null,
  templateName: '',
  templateDescription: '',
};

let designerState: DesignerState = { ...initialState };

const listeners = new Set<(state: DesignerState) => void>();

export function generateId(prefix: string = 'node'): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

const getDefaultNodeData = (type: NodeType): DesignerNodeData => {
  switch (type) {
    case 'vm':
      return {
        type: 'vm',
        name: 'new-vm',
        vcpuCount: 2,
        memoryMib: 512,
        diskSizeGb: 10,
        image: 'ubuntu-22.04',
      } as VMNodeData;
    case 'network':
      return {
        type: 'network',
        name: 'new-network',
        cidr: '10.0.0.0/24',
        networkType: 'bridge',
      } as NetworkNodeData;
    case 'volume':
      return {
        type: 'volume',
        name: 'new-volume',
        sizeGb: 10,
        isPersistent: true,
      } as VolumeNodeData;
    default:
      throw new Error(`Unknown node type: ${type}`);
  }
};

function notify() {
  listeners.forEach((l) => l(designerState));
}

export const designerStore = {
  // State access
  getState: () => designerState,
  subscribe: (listener: (state: DesignerState) => void) => {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },

  // Actions
  addNode: (type: NodeType, position: { x: number; y: number }) => {
    const newNode: DesignerNode = {
      id: generateId(type),
      type,
      position,
      data: getDefaultNodeData(type),
    };
    designerState = {
      ...designerState,
      nodes: [...designerState.nodes, newNode],
    };
    notify();
  },

  removeNode: (nodeId: string) => {
    designerState = {
      ...designerState,
      nodes: designerState.nodes.filter((n) => n.id !== nodeId),
      edges: designerState.edges.filter(
        (e) => e.source !== nodeId && e.target !== nodeId
      ),
      selectedNodeId: designerState.selectedNodeId === nodeId ? null : designerState.selectedNodeId,
    };
    notify();
  },

  updateNodeData: (nodeId: string, data: Partial<DesignerNodeData>) => {
    designerState = {
      ...designerState,
      nodes: designerState.nodes.map((n) =>
        n.id === nodeId ? { ...n, data: { ...n.data, ...data } as DesignerNodeData } : n
      ),
    };
    notify();
  },

  selectNode: (nodeId: string | null) => {
    designerState = {
      ...designerState,
      selectedNodeId: nodeId,
    };
    notify();
  },

  setNodes: (nodes: DesignerNode[]) => {
    designerState = { ...designerState, nodes };
    notify();
  },

  setEdges: (edges: DesignerEdge[]) => {
    designerState = { ...designerState, edges };
    notify();
  },

  setTemplateName: (name: string) => {
    designerState = { ...designerState, templateName: name };
    notify();
  },

  setTemplateDescription: (description: string) => {
    designerState = { ...designerState, templateDescription: description };
    notify();
  },

  toTemplate: (): DesignerTemplate => ({
    id: generateId('template'),
    name: designerState.templateName,
    description: designerState.templateDescription,
    nodes: designerState.nodes,
    edges: designerState.edges,
    version: '1.0.0',
  }),

  clearCanvas: () => {
    designerState = { ...initialState };
    notify();
  },
};

// React hook for using the designer store
export function useDesignerStore() {
  const state = useSyncExternalStore(
    designerStore.subscribe,
    designerStore.getState,
    designerStore.getState
  );

  const selectedNode = state.nodes.find((n) => n.id === state.selectedNodeId) || null;

  return {
    nodes: state.nodes,
    edges: state.edges,
    selectedNodeId: state.selectedNodeId,
    selectedNode,
    templateName: state.templateName,
    templateDescription: state.templateDescription,
    // Actions
    addNode: designerStore.addNode,
    removeNode: designerStore.removeNode,
    updateNodeData: designerStore.updateNodeData,
    selectNode: designerStore.selectNode,
    setNodes: designerStore.setNodes,
    setEdges: designerStore.setEdges,
    setTemplateName: designerStore.setTemplateName,
    setTemplateDescription: designerStore.setTemplateDescription,
    toTemplate: designerStore.toTemplate,
    clearCanvas: designerStore.clearCanvas,
  };
}
