import type {
  DesignerNode,
  DesignerEdge,
  DesignerNodeType,
  VMNodeData,
  NetworkNodeData,
  VolumeNodeData,
  AnyNodeData,
} from '@/types/designer';

/**
 * Generate a unique ID with the given prefix
 */
export function generateId(prefix: string = 'node'): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Create default data for a VM node
 */
export function createDefaultVMData(): VMNodeData {
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

/**
 * Create default data for a Network node
 */
export function createDefaultNetworkData(): NetworkNodeData {
  return {
    label: 'Network',
    type: 'network',
    name: 'new-network',
    networkType: 'bridge',
  };
}

/**
 * Create default data for a Volume node
 */
export function createDefaultVolumeData(): VolumeNodeData {
  return {
    label: 'Volume',
    type: 'volume',
    name: 'new-volume',
    sizeGb: 10,
    format: 'raw',
    isPersistent: true,
  };
}

/**
 * Create default node data based on node type
 */
export function createDefaultNodeData(type: DesignerNodeType): AnyNodeData {
  switch (type) {
    case 'vm':
      return createDefaultVMData();
    case 'network':
      return createDefaultNetworkData();
    case 'volume':
      return createDefaultVolumeData();
    default:
      throw new Error(`Unknown node type: ${type}`);
  }
}

/**
 * Execution plan step types
 */
export type ExecutionStepType = 'create_network' | 'create_volume' | 'create_vm' | 'configure_network';

/**
 * Execution plan step
 */
export interface ExecutionStep {
  id: string;
  type: ExecutionStepType;
  name: string;
  dependsOn: string[];
  config: Record<string, unknown>;
}

/**
 * Convert designer state to an ordered execution plan
 * Orders: networks first, then volumes, then VMs
 */
export function convertToExecutionPlan(nodes: DesignerNode[], edges: DesignerEdge[]): ExecutionStep[] {
  const steps: ExecutionStep[] = [];
  const nodeMap = new Map(nodes.map((n) => [n.id, n]));

  // Build dependency graph from edges
  const dependencies = new Map<string, string[]>();
  edges.forEach((edge) => {
    if (!dependencies.has(edge.target)) {
      dependencies.set(edge.target, []);
    }
    dependencies.get(edge.target)!.push(edge.source);
  });

  // Helper to get execution step type from node type
  const getStepType = (node: DesignerNode): ExecutionStepType => {
    switch (node.type) {
      case 'network':
        return 'create_network';
      case 'volume':
        return 'create_volume';
      case 'vm':
        return 'create_vm';
      default:
        throw new Error(`Unknown node type: ${node.type}`);
    }
  };

  // Helper to build config from node data
  const buildConfig = (node: DesignerNode): Record<string, unknown> => {
    const { label, type, ...config } = node.data;
    return {
      ...config,
      name: node.data.name,
    };
  };

  // Create steps for all nodes
  nodes.forEach((node) => {
    steps.push({
      id: node.id,
      type: getStepType(node),
      name: (node.data.name as string) || `${node.type}-${node.id}`,
      dependsOn: dependencies.get(node.id) || [],
      config: buildConfig(node),
    });
  });

  // Sort by dependencies (topological sort)
  const sorted: ExecutionStep[] = [];
  const visited = new Set<string>();
  const visiting = new Set<string>();

  const visit = (step: ExecutionStep) => {
    if (visited.has(step.id)) return;
    if (visiting.has(step.id)) {
      throw new Error(`Circular dependency detected at node: ${step.id}`);
    }

    visiting.add(step.id);
    
    // Visit dependencies first
    for (const depId of step.dependsOn) {
      const depStep = steps.find((s) => s.id === depId);
      if (depStep) {
        visit(depStep);
      }
    }

    visiting.delete(step.id);
    visited.add(step.id);
    sorted.push(step);
  };

  // Visit all steps
  steps.forEach((step) => visit(step));

  return sorted;
}

/**
 * Validation error
 */
export interface ValidationError {
  nodeId?: string;
  message: string;
}

/**
 * Validation result
 */
export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * Validate a template for deployment
 */
export function validateTemplate(nodes: DesignerNode[], edges: DesignerEdge[]): ValidationResult {
  const errors: string[] = [];

  // Check for empty canvas
  if (nodes.length === 0) {
    errors.push('Template must contain at least one node');
  }

  // Validate each node
  nodes.forEach((node) => {
    // Check for name
    const name = node.data.name as string | undefined;
    if (!name || name.trim() === '') {
      errors.push(`Node ${node.id} (${node.type}) is missing a name`);
    }

    // Type-specific validation
    switch (node.type) {
      case 'vm': {
        const vmData = node.data as VMNodeData;
        if (!vmData.vcpuCount || vmData.vcpuCount < 1) {
          errors.push(`VM "${name}" must have at least 1 vCPU`);
        }
        if (!vmData.memoryMib || vmData.memoryMib < 64) {
          errors.push(`VM "${name}" must have at least 64 MiB of memory`);
        }
        break;
      }

      case 'network': {
        const netData = node.data as NetworkNodeData;
        if (!netData.networkType) {
          errors.push(`Network "${name}" must have a network type`);
        }
        break;
      }

      case 'volume': {
        const volData = node.data as VolumeNodeData;
        if (!volData.sizeGb || volData.sizeGb < 1) {
          errors.push(`Volume "${name}" must have a size of at least 1 GB`);
        }
        break;
      }
    }
  });

  // Check for orphaned nodes (not necessarily an error, but worth noting)
  const connectedNodeIds = new Set<string>();
  edges.forEach((e) => {
    connectedNodeIds.add(e.source);
    connectedNodeIds.add(e.target);
  });

  const orphanedNodes = nodes.filter((n) => !connectedNodeIds.has(n.id));
  if (orphanedNodes.length > 0 && nodes.length > 1) {
    // This is a warning, not an error
    console.warn(
      'Warning: Some nodes are not connected to any other nodes:',
      orphanedNodes.map((n) => n.data.name || n.id)
    );
  }

  // Check for duplicate names
  const names = new Map<string, string>();
  nodes.forEach((node) => {
    const nodeName = node.data.name as string | undefined;
    if (nodeName) {
      if (names.has(nodeName)) {
        errors.push(`Duplicate name "${nodeName}" found. Names must be unique.`);
      } else {
        names.set(nodeName, node.id);
      }
    }
  });

  return {
    valid: errors.length === 0,
    errors,
  };
}

/**
 * Export template to JSON string
 */
export function exportTemplateToJson(
  nodes: DesignerNode[],
  edges: DesignerEdge[],
  metadata: { name: string; description: string }
): string {
  const template = {
    id: generateId('template'),
    name: metadata.name,
    description: metadata.description,
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

  return JSON.stringify(template, null, 2);
}

/**
 * Import template from JSON string
 */
export function importTemplateFromJson(json: string): {
  nodes: DesignerNode[];
  edges: DesignerEdge[];
  metadata: { name: string; description: string };
} | null {
  try {
    const template = JSON.parse(json);
    
    return {
      nodes: template.nodes || [],
      edges: template.edges || [],
      metadata: {
        name: template.name || 'Imported Template',
        description: template.description || '',
      },
    };
  } catch (e) {
    console.error('Failed to parse template JSON:', e);
    return null;
  }
}

/**
 * Get node color based on type (for UI)
 */
export function getNodeColor(type: DesignerNodeType): string {
  switch (type) {
    case 'vm':
      return '#3b82f6'; // blue-500
    case 'network':
      return '#10b981'; // emerald-500
    case 'volume':
      return '#f59e0b'; // amber-500
    default:
      return '#6b7280'; // gray-500
  }
}

/**
 * Get node icon based on type (for UI)
 */
export function getNodeIcon(type: DesignerNodeType): string {
  switch (type) {
    case 'vm':
      return 'Server';
    case 'network':
      return 'Network';
    case 'volume':
      return 'HardDrive';
    default:
      return 'Box';
  }
}

/**
 * Calculate canvas bounds for all nodes
 */
export function calculateCanvasBounds(
  nodes: DesignerNode[],
  padding: number = 50
): { minX: number; minY: number; maxX: number; maxY: number; width: number; height: number } | null {
  if (nodes.length === 0) return null;

  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;

  nodes.forEach((node) => {
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
    maxX = Math.max(maxX, node.position.x);
    maxY = Math.max(maxY, node.position.y);
  });

  return {
    minX: minX - padding,
    minY: minY - padding,
    maxX: maxX + padding,
    maxY: maxY + padding,
    width: maxX - minX + padding * 2,
    height: maxY - minY + padding * 2,
  };
}

/**
 * Auto-layout nodes in a grid pattern
 */
export function autoLayoutNodes(
  nodes: DesignerNode[],
  gridSize: number = 200
): DesignerNode[] {
  const cols = Math.ceil(Math.sqrt(nodes.length));
  
  return nodes.map((node, index) => ({
    ...node,
    position: {
      x: (index % cols) * gridSize + 100,
      y: Math.floor(index / cols) * gridSize + 100,
    },
  }));
}
