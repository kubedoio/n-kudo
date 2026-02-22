import { DesignerNode, DesignerEdge, DesignerTemplate, DesignerNodeType, Template } from '@/types/designer';
import { Connection, NodeChange, EdgeChange, applyNodeChanges, applyEdgeChanges } from '@xyflow/react';
import { useSyncExternalStore } from 'react';

export interface DesignerState {
    // Canvas state
    nodes: DesignerNode[];
    edges: DesignerEdge[];
    selectedNodeId: string | null;

    // Template metadata
    templateName: string;
    templateDescription: string;
    templateId: string | null;
    
    // UI state
    isDirty: boolean;
    isDeploying: boolean;
    currentTemplate: Template | null;
}

const initialState: DesignerState = {
    nodes: [],
    edges: [],
    selectedNodeId: null,
    templateName: '',
    templateDescription: '',
    templateId: null,
    isDirty: false,
    isDeploying: false,
    currentTemplate: null,
};

let designerState: DesignerState = { ...initialState };

const listeners = new Set<(state: DesignerState) => void>();

export function generateId(prefix: string = 'node'): string {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

const getDefaultNodeData = (type: DesignerNodeType) => {
    switch (type) {
        case 'vm':
            return {
                label: 'VM',
                type: 'vm' as const,
                name: 'new-vm',
                vcpuCount: 2,
                memoryMib: 512,
                kernel: '',
                rootfs: '',
                networkInterfaces: [],
                volumes: [],
            };
        case 'network':
            return {
                label: 'Network',
                type: 'network' as const,
                name: 'new-network',
                networkType: 'bridge' as const,
            };
        case 'volume':
            return {
                label: 'Volume',
                type: 'volume' as const,
                name: 'new-volume',
                sizeGb: 10,
                format: 'raw' as const,
                isPersistent: true,
            };
        default:
            return { label: 'Node', type };
    }
};

export const defaultNodeData: Record<DesignerNodeType, ReturnType<typeof getDefaultNodeData>> = {
    vm: getDefaultNodeData('vm'),
    network: getDefaultNodeData('network'),
    volume: getDefaultNodeData('volume'),
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
    addNode: (node: DesignerNode | { type: DesignerNodeType; position: { x: number; y: number } }) => {
        if ('id' in node) {
            // Adding a full node
            designerState = {
                ...designerState,
                nodes: [...designerState.nodes, node as DesignerNode],
                isDirty: true,
            };
        } else {
            // Adding from type and position
            const newNode: DesignerNode = {
                id: generateId(node.type),
                type: node.type,
                position: node.position,
                data: getDefaultNodeData(node.type),
            };
            designerState = {
                ...designerState,
                nodes: [...designerState.nodes, newNode],
                isDirty: true,
            };
        }
        notify();
    },

    addEdge: (edge: DesignerEdge | { id?: string; source: string; target: string; label?: string }) => {
        const newEdge: DesignerEdge = {
            id: edge.id || generateId('edge'),
            source: edge.source,
            target: edge.target,
            ...(edge as any).sourceHandle && { sourceHandle: (edge as any).sourceHandle },
            ...(edge as any).targetHandle && { targetHandle: (edge as any).targetHandle },
            ...(edge as any).label && { label: (edge as any).label },
        };
        designerState = {
            ...designerState,
            edges: [...designerState.edges, newEdge],
            isDirty: true,
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
            isDirty: true,
        };
        notify();
    },

    updateNodeData: (nodeId: string, data: Partial<DesignerNode['data']>) => {
        designerState = {
            ...designerState,
            nodes: designerState.nodes.map((n) =>
                n.id === nodeId ? { ...n, data: { ...n.data, ...data } } : n
            ),
            isDirty: true,
        };
        notify();
    },

    setNodes: (nodes: DesignerNode[]) => {
        designerState = { ...designerState, nodes, isDirty: true };
        notify();
    },

    setEdges: (edges: DesignerEdge[]) => {
        designerState = { ...designerState, edges, isDirty: true };
        notify();
    },

    onNodesChange: (changes: NodeChange[]) => {
        designerState = {
            ...designerState,
            nodes: applyNodeChanges(changes, designerState.nodes) as DesignerNode[],
        };
        // Track selection changes
        const selectionChange = changes.find((c) => c.type === 'select');
        if (selectionChange && 'id' in selectionChange) {
            const isSelected = (selectionChange as any).selected;
            designerState = {
                ...designerState,
                selectedNodeId: isSelected ? (selectionChange as any).id : null,
            };
        }
        notify();
    },

    onEdgesChange: (changes: EdgeChange[]) => {
        designerState = {
            ...designerState,
            edges: applyEdgeChanges(changes, designerState.edges) as DesignerEdge[],
        };
        notify();
    },

    onConnect: (connection: Connection) => {
        if (!connection.source || !connection.target) return;

        const newEdge: DesignerEdge = {
            id: `e-${connection.source}-${connection.target}`,
            source: connection.source,
            target: connection.target,
            sourceHandle: connection.sourceHandle || undefined,
            targetHandle: connection.targetHandle || undefined,
        };
        designerState = {
            ...designerState,
            edges: [...designerState.edges, newEdge],
            isDirty: true,
        };
        notify();
    },

    selectNode: (node: DesignerNode | null) => {
        designerState = {
            ...designerState,
            selectedNodeId: node?.id || null,
            nodes: designerState.nodes.map((n) => ({
                ...n,
                selected: n.id === node?.id,
            })),
        };
        notify();
    },

    setSelectedNode: (nodeId: string | null) => {
        designerState = {
            ...designerState,
            selectedNodeId: nodeId,
            nodes: designerState.nodes.map((n) => ({
                ...n,
                selected: n.id === nodeId,
            })),
        };
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

    setCurrentTemplate: (template: Template | null) => {
        designerState = {
            ...designerState,
            currentTemplate: template,
            isDirty: false,
        };
        notify();
    },

    setDeploying: (isDeploying: boolean) => {
        designerState = { ...designerState, isDeploying };
        notify();
    },

    markDirty: () => {
        designerState = { ...designerState, isDirty: true };
        notify();
    },

    loadTemplate: (template: Template | DesignerTemplate) => {
        const nodes = 'nodes' in template ? template.nodes : [];
        const edges = 'edges' in template ? template.edges : [];
        const name = 'name' in template ? template.name : '';
        const description = 'description' in template ? template.description : '';
        const id = 'id' in template ? template.id || null : null;
        
        designerState = {
            ...designerState,
            nodes: nodes as DesignerNode[],
            edges: edges as DesignerEdge[],
            templateName: name,
            templateDescription: description,
            templateId: id,
            selectedNodeId: null,

            isDirty: false,
        };
        notify();
    },

    toTemplate: (): DesignerTemplate => ({
        id: designerState.templateId || undefined,
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

    return {
        nodes: state.nodes,
        edges: state.edges,
        selectedNode: state.nodes.find((n) => n.id === state.selectedNodeId) || null,
        selectedNodeId: state.selectedNodeId,
        templateName: state.templateName,
        templateDescription: state.templateDescription,
        templateId: state.templateId,
        // Additional UI state
        isDirty: state.isDirty,
        isDeploying: state.isDeploying,
        currentTemplate: state.templateId ? {
            id: state.templateId,
            name: state.templateName,
            description: state.templateDescription,
            nodes: state.nodes,
            edges: state.edges,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        } : null,
        // Actions
        setNodes: designerStore.setNodes,
        setEdges: designerStore.setEdges,
        selectNode: designerStore.selectNode,
        setSelectedNode: designerStore.setSelectedNode,
        addNode: designerStore.addNode,
        addEdge: designerStore.addEdge,
        removeNode: designerStore.removeNode,
        updateNodeData: designerStore.updateNodeData,
        onNodesChange: designerStore.onNodesChange,
        onEdgesChange: designerStore.onEdgesChange,
        onConnect: designerStore.onConnect,
        setTemplateName: designerStore.setTemplateName,
        setTemplateDescription: designerStore.setTemplateDescription,
        setCurrentTemplate: designerStore.setCurrentTemplate,
        setDeploying: designerStore.setDeploying,
        markDirty: designerStore.markDirty,
        loadTemplate: designerStore.loadTemplate,
        toTemplate: designerStore.toTemplate,
        clearCanvas: designerStore.clearCanvas,
    };
}
