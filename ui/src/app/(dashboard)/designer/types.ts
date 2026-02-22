/**
 * Designer Types - Type definitions for the visual designer feature
 * MVP-2 Phase 1
 * 
 * These types extend the global designer types with additional fields needed
 * for the property panel forms.
 */

import {
  VMNodeData as GlobalVMNodeData,
  NetworkNodeData as GlobalNetworkNodeData,
  VolumeNodeData as GlobalVolumeNodeData,
  DesignerNode as GlobalDesignerNode,
  DesignerEdge as GlobalDesignerEdge,
  DesignerTemplate as GlobalDesignerTemplate,
  DesignerNodeType,
} from '@/types/designer';

export type NodeType = DesignerNodeType;

// Extend global VMNodeData with additional fields needed for forms
export interface VMNodeData extends GlobalVMNodeData {
  memoryMib: number;
  diskSizeGb: number;
  image: 'ubuntu-22.04' | 'debian-12' | 'alpine-3.19';
}

// Extend global NetworkNodeData with additional fields needed for forms
export interface NetworkNodeData extends GlobalNetworkNodeData {
  networkType: 'bridge' | 'vxlan';
}

// Extend global VolumeNodeData with additional fields needed for forms
export interface VolumeNodeData extends GlobalVolumeNodeData {
  sizeGb: number;
  isPersistent: boolean;
}

export type DesignerNodeData = VMNodeData | NetworkNodeData | VolumeNodeData;

export interface DesignerNode extends GlobalDesignerNode {
  data: DesignerNodeData;
}

export interface DesignerEdge extends GlobalDesignerEdge {}

export interface Template {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  nodes: any[]; // ReactFlow nodes
  edges: any[]; // ReactFlow edges
  createdAt: string;
  updatedAt: string;
}

export interface DesignerTemplate extends GlobalDesignerTemplate {
  nodes: DesignerNode[];
  edges: DesignerEdge[];
}
