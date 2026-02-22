import { Node, Edge } from '@xyflow/react';

/**
 * Designer Types - Type definitions for the visual designer feature
 */

// Node types supported by the designer
export type DesignerNodeType = 'vm' | 'network' | 'volume';

// Export as object for runtime use
export const DesignerNodeType = {
    VM: 'vm',
    NETWORK: 'network',
    VOLUME: 'volume',
} as const;

// Base interface for all node data
export interface DesignerNodeData extends Record<string, unknown> {
    label: string;
    type: DesignerNodeType;
}

// VM Node data
export interface VMNodeData extends DesignerNodeData {
    type: 'vm';
    name: string;
    vcpuCount: number;
    memoryMib: number;
    kernel?: string;
    rootfs?: string;
    networkInterfaces?: string[];
    volumes?: string[];
}

// Network Node data
export interface NetworkNodeData extends DesignerNodeData {
    type: 'network';
    name: string;
    networkType: 'bridge' | 'vxlan' | 'tap';
    cidr?: string;
    vni?: number;
    parentInterface?: string;
}

// Volume/Storage Node data
export interface VolumeNodeData extends DesignerNodeData {
    type: 'volume';
    name: string;
    sizeGb: number;
    format: 'raw' | 'qcow2';
    isPersistent: boolean;
}

// Union type for all node data
export type AnyNodeData = VMNodeData | NetworkNodeData | VolumeNodeData;

// React Flow compatible node type
export type DesignerNode = Node<DesignerNodeData>;

// React Flow compatible edge type
export type DesignerEdge = Edge;

// Template definition for the designer (client-side)
export interface DesignerTemplate {
    id?: string;
    name: string;
    description: string;
    nodes: DesignerNode[];
    edges: DesignerEdge[];
    version: string;
}

// Legacy Template interface for backward compatibility
export interface Template {
    id: string;
    name: string;
    description: string;
    nodes: TemplateNode[];
    edges: TemplateEdge[];
    createdAt: string;
    updatedAt: string;
}

// Simplified node for templates
export interface TemplateNode {
    id: string;
    type: DesignerNodeType;
    position: { x: number; y: number };
    data: AnyNodeData;
}

// Simplified edge for templates
export interface TemplateEdge {
    id: string;
    source: string;
    target: string;
    label?: string;
}

// Palette item definition
export interface PaletteItem {
    type: DesignerNodeType;
    label: string;
    description: string;
    icon: string;
    category: 'compute' | 'networking' | 'storage';
    defaultData: Partial<AnyNodeData>;
}
