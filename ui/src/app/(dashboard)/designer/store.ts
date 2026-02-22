import { create } from 'zustand';
import { devtools, subscribeWithSelector } from 'zustand/middleware';
import type {
  Template,
  DesignerNode,
  DesignerEdge,
  DesignerNodeType,
  VMNodeData,
  NetworkNodeData,
  VolumeNodeData,
  DesignerTemplate,
  AnyNodeData,
} from '@/types/designer';
import { applyNodeChanges, applyEdgeChanges, type NodeChange, type EdgeChange, type Connection } from '@xyflow/react';
import { generateId } from './utils';

export interface DesignerState {
  // Canvas state
  nodes: DesignerNode[];
  edges: DesignerEdge[];
  selectedNodeId: string | null;

  // Template metadata
  currentTemplate: Template | null;
  templateName: string;
  templateDescription: string;
  templateId: string | null;
  isDirty: boolean;

  // Actions
  setNodes: (nodes: DesignerNode[] | ((prev: DesignerNode[]) => DesignerNode[])) => void;
  setEdges: (edges: DesignerEdge[] | ((prev: DesignerEdge[]) => DesignerEdge[])) => void;
  onNodesChange: (changes: NodeChange[]) => void;
  onEdgesChange: (changes: EdgeChange[]) => void;
  selectNode: (id: string | null) => void;
  addNode: (type: DesignerNodeType, position: { x: number; y: number }) => void;
  updateNodeData: (id: string, data: Partial<DesignerNode['data']>) => void;
  removeNode: (id: string) => void;
  addEdge: (connection: Connection) => void;
  removeEdge: (id: string) => void;
  setCurrentTemplate: (template: Template | null) => void;
  setTemplateName: (name: string) => void;
  setTemplateDescription: (description: string) => void;
  markDirty: (dirty: boolean) => void;
  reset: () => void;

  // Template operations
  createTemplate: (name: string, description: string) => Template;
  loadTemplate: (template: Template | DesignerTemplate) => void;
  toTemplate: () => DesignerTemplate;
  clearCanvas: () => void;
}

// Default node data creators
function getDefaultVMData(): VMNodeData {
  return {
    label: 'VM',
    type: 'vm',
    name: 'new-vm',
    vcpuCount: 2,
    memoryMib: 512,
    kernel: '',
    rootfs: '',
    networkInterfaces: [],
    volumes: [],
  };
}

function getDefaultNetworkData(): NetworkNodeData {
  return {
    label: 'Network',
    type: 'network',
    name: 'new-network',
    networkType: 'bridge',
  };
}

function getDefaultVolumeData(): VolumeNodeData {
  return {
    label: 'Volume',
    type: 'volume',
    name: 'new-volume',
    sizeGb: 10,
    format: 'raw',
    isPersistent: true,
  };
}

function getDefaultNodeData(type: DesignerNodeType): VMNodeData | NetworkNodeData | VolumeNodeData {
  switch (type) {
    case 'vm':
      return getDefaultVMData();
    case 'network':
      return getDefaultNetworkData();
    case 'volume':
      return getDefaultVolumeData();
    default:
      throw new Error(`Unknown node type: ${type}`);
  }
}

const initialState = {
  nodes: [] as DesignerNode[],
  edges: [] as DesignerEdge[],
  selectedNodeId: null as string | null,
  currentTemplate: null as Template | null,
  templateName: '',
  templateDescription: '',
  templateId: null as string | null,
  isDirty: false,
};

export const useDesignerStore = create<DesignerState>()(
  devtools(
    subscribeWithSelector((set, get) => ({
      ...initialState,

      // Set nodes directly or via function
      setNodes: (nodes) => {
        set((state) => ({
          nodes: typeof nodes === 'function' ? nodes(state.nodes) : nodes,
          isDirty: true,
        }));
      },

      // Set edges directly or via function
      setEdges: (edges) => {
        set((state) => ({
          edges: typeof edges === 'function' ? edges(state.edges) : edges,
          isDirty: true,
        }));
      },

      // Handle node changes from React Flow (selection, position, etc.)
      onNodesChange: (changes) => {
        set((state) => {
          const updatedNodes = applyNodeChanges(changes, state.nodes) as DesignerNode[];
          
          // Track selection changes
          const selectionChange = changes.find((c) => c.type === 'select');
          const selectedNodeId = selectionChange && 'id' in selectionChange
            ? (selectionChange as any).selected
              ? (selectionChange as any).id
              : null
            : state.selectedNodeId;

          return {
            nodes: updatedNodes,
            selectedNodeId,
          };
        });
      },

      // Handle edge changes from React Flow
      onEdgesChange: (changes) => {
        set((state) => ({
          edges: applyEdgeChanges(changes, state.edges) as DesignerEdge[],
        }));
      },

      // Select a node by ID
      selectNode: (id) => {
        set((state) => ({
          selectedNodeId: id,
          nodes: state.nodes.map((n) => ({
            ...n,
            selected: n.id === id,
          })),
        }));
      },

      // Add a new node
      addNode: (type, position) => {
        const newNode: DesignerNode = {
          id: generateId(type),
          type,
          position,
          data: getDefaultNodeData(type),
        };

        set((state) => ({
          nodes: [...state.nodes, newNode],
          isDirty: true,
        }));
      },

      // Update node data
      updateNodeData: (id, data) => {
        set((state) => ({
          nodes: state.nodes.map((n) =>
            n.id === id ? { ...n, data: { ...n.data, ...data } } : n
          ),
          isDirty: true,
        }));
      },

      // Remove a node and its connected edges
      removeNode: (id) => {
        set((state) => ({
          nodes: state.nodes.filter((n) => n.id !== id),
          edges: state.edges.filter((e) => e.source !== id && e.target !== id),
          selectedNodeId: state.selectedNodeId === id ? null : state.selectedNodeId,
          isDirty: true,
        }));
      },

      // Add an edge from a connection
      addEdge: (connection) => {
        if (!connection.source || !connection.target) return;

        const newEdge: DesignerEdge = {
          id: `e-${connection.source}-${connection.target}`,
          source: connection.source,
          target: connection.target,
          sourceHandle: connection.sourceHandle || undefined,
          targetHandle: connection.targetHandle || undefined,
        };

        set((state) => ({
          edges: [...state.edges, newEdge],
          isDirty: true,
        }));
      },

      // Remove an edge by ID
      removeEdge: (id) => {
        set((state) => ({
          edges: state.edges.filter((e) => e.id !== id),
          isDirty: true,
        }));
      },

      // Set current template
      setCurrentTemplate: (template) => {
        set({
          currentTemplate: template,
          isDirty: false,
        });
      },

      // Set template name
      setTemplateName: (name) => {
        set({ templateName: name });
      },

      // Set template description
      setTemplateDescription: (description) => {
        set({ templateDescription: description });
      },

      // Mark canvas as dirty/clean
      markDirty: (dirty) => {
        set({ isDirty: dirty });
      },

      // Reset to initial state
      reset: () => {
        set(initialState);
      },

      // Create a template from current state
      createTemplate: (name, description) => {
        const state = get();
        const template: Template = {
          id: generateId('template'),
          name,
          description,
          nodes: state.nodes.map((n) => ({
            id: n.id,
            type: n.type as DesignerNodeType,
            position: n.position,
            data: n.data as AnyNodeData,
          })),
          edges: state.edges.map((e) => ({
            id: e.id,
            source: e.source,
            target: e.target,
            label: typeof e.label === 'string' ? e.label : undefined,
          })),
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };

        set({
          currentTemplate: template,
          templateName: name,
          templateDescription: description,
          templateId: template.id,
          isDirty: false,
        });

        return template;
      },

      // Load a template into the canvas
      loadTemplate: (template) => {
        const nodes = 'nodes' in template ? (template as any).nodes : [];
        const edges = 'edges' in template ? (template as any).edges : [];
        const name = 'name' in template ? (template as any).name : '';
        const description = 'description' in template ? (template as any).description : '';
        const id = 'id' in template ? (template as any).id : null;

        set({
          nodes: nodes as DesignerNode[],
          edges: edges as DesignerEdge[],
          templateName: name,
          templateDescription: description,
          templateId: id,
          selectedNodeId: null,
          isDirty: false,
        });
      },

      // Convert current state to DesignerTemplate
      toTemplate: () => {
        const state = get();
        return {
          id: state.templateId || undefined,
          name: state.templateName,
          description: state.templateDescription,
          nodes: state.nodes,
          edges: state.edges,
          version: '1.0.0',
        };
      },

      // Clear canvas
      clearCanvas: () => {
        set({
          nodes: [],
          edges: [],
          selectedNodeId: null,
          templateName: '',
          templateDescription: '',
          templateId: null,
          isDirty: false,
        });
      },
    })),
    { name: 'designer-store' }
  )
);

// Export selector hooks for performance
export const selectNodes = (state: DesignerState) => state.nodes;
export const selectEdges = (state: DesignerState) => state.edges;
export const selectSelectedNodeId = (state: DesignerState) => state.selectedNodeId;
export const selectSelectedNode = (state: DesignerState) =>
  state.nodes.find((n) => n.id === state.selectedNodeId) || null;
export const selectIsDirty = (state: DesignerState) => state.isDirty;
export const selectCurrentTemplate = (state: DesignerState) => state.currentTemplate;
