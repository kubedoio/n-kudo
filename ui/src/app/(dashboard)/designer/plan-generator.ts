/**
 * Plan Generator - Converts designer canvas to execution actions
 * MVP-2 Phase 7
 */

export interface PlanAction {
  id: string;
  operation: 'CREATE_VM' | 'CREATE_NETWORK' | 'CREATE_VOLUME' | 'ATTACH_NETWORK' | 'ATTACH_VOLUME';
  resource_type: string;
  params: Record<string, any>;
  dependencies?: string[];
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

/**
 * Generate a plan (list of actions) from designer nodes and edges
 */
export function generatePlanFromDesign(nodes: any[], edges: any[]): PlanAction[] {
  const actions: PlanAction[] = [];
  const createdResources: Map<string, string> = new Map(); // nodeId -> resourceId

  // First pass: Create independent resources (volumes, networks)
  nodes.forEach((node) => {
    if (node.data?.type === 'volume') {
      const action: PlanAction = {
        id: `action-volume-${node.id}`,
        operation: 'CREATE_VOLUME',
        resource_type: 'volume',
        params: {
          name: node.data.name,
          size_gib: node.data.sizeGb || node.data.sizeGiB || 10,
          persistent: node.data.isPersistent ?? node.data.persistent ?? true,
        },
      };
      actions.push(action);
      createdResources.set(node.id, `volume:${node.data.name}`);
    }

    if (node.data?.type === 'network') {
      const action: PlanAction = {
        id: `action-network-${node.id}`,
        operation: 'CREATE_NETWORK',
        resource_type: 'network',
        params: {
          name: node.data.name,
          cidr: node.data.cidr,
          bridge_name: node.data.bridgeName,
        },
      };
      actions.push(action);
      createdResources.set(node.id, `network:${node.data.name}`);
    }
  });

  // Second pass: Create VMs with dependencies
  nodes.forEach((node) => {
    if (node.data?.type === 'vm') {
      // Find connected networks and volumes
      const connectedEdges = edges.filter(
        (e) => e.source === node.id || e.target === node.id
      );

      const networkIds = connectedEdges
        .map((e) => (e.source === node.id ? e.target : e.source))
        .filter((id) => nodes.find((n) => n.id === id)?.data?.type === 'network');

      const volumeIds = connectedEdges
        .map((e) => (e.source === node.id ? e.target : e.source))
        .filter((id) => nodes.find((n) => n.id === id)?.data?.type === 'volume');

      const action: PlanAction = {
        id: `action-vm-${node.id}`,
        operation: 'CREATE_VM',
        resource_type: 'microvm',
        params: {
          name: node.data.name,
          vcpu_count: node.data.vcpuCount || 2,
          memory_mib: node.data.memoryMib || node.data.memoryMiB || 512,
          disk_gib: node.data.diskGiB || 10,
          image: node.data.image || node.data.kernel || 'ubuntu-22.04',
          network_ids: networkIds.map((id) => createdResources.get(id)).filter(Boolean),
          volume_ids: volumeIds.map((id) => createdResources.get(id)).filter(Boolean),
        },
        dependencies: [...networkIds, ...volumeIds]
          .map((id) => createdResources.get(id))
          .filter(Boolean) as string[],
      };
      actions.push(action);
    }
  });

  return actions;
}

/**
 * Validate the design for common issues
 */
export function validateDesign(nodes: any[], edges: any[]): ValidationResult {
  const errors: string[] = [];

  // Check for empty canvas
  if (nodes.length === 0) {
    errors.push('Design is empty. Add at least one component.');
    return { valid: false, errors };
  }

  // Check for duplicate names
  const names = nodes.map((n) => n.data?.name).filter(Boolean);
  const duplicates = names.filter((item, index) => names.indexOf(item) !== index);
  if (duplicates.length > 0) {
    errors.push(`Duplicate names found: ${[...new Set(duplicates)].join(', ')}`);
  }

  // Check for empty names
  const emptyNames = nodes.filter((n) => !n.data?.name?.trim());
  if (emptyNames.length > 0) {
    errors.push('All components must have a name');
  }

  // Check each VM is connected to at least one network
  const vms = nodes.filter((n) => n.data?.type === 'vm');
  vms.forEach((vm) => {
    const hasNetwork = edges.some(
      (e) =>
        (e.source === vm.id &&
          nodes.find((n) => n.id === e.target)?.data?.type === 'network') ||
        (e.target === vm.id &&
          nodes.find((n) => n.id === e.source)?.data?.type === 'network')
    );
    if (!hasNetwork) {
      errors.push(`VM "${vm.data.name}" must be connected to a network`);
    }
  });

  // Validate network CIDRs
  const networks = nodes.filter((n) => n.data?.type === 'network');
  networks.forEach((network) => {
    if (network.data.cidr && !isValidCIDR(network.data.cidr)) {
      errors.push(`Network "${network.data.name}" has invalid CIDR: ${network.data.cidr}`);
    }
  });

  // Validate VM specs
  vms.forEach((vm) => {
    const vcpuCount = vm.data.vcpuCount || 0;
    const memoryMib = vm.data.memoryMib || vm.data.memoryMiB || 0;

    if (vcpuCount < 1) {
      errors.push(`VM "${vm.data.name}" must have at least 1 vCPU`);
    }
    if (memoryMib < 64) {
      errors.push(`VM "${vm.data.name}" must have at least 64 MiB of memory`);
    }
  });

  // Validate volume sizes
  const volumes = nodes.filter((n) => n.data?.type === 'volume');
  volumes.forEach((volume) => {
    const size = volume.data.sizeGb || volume.data.sizeGiB || 0;
    if (size < 1) {
      errors.push(`Volume "${volume.data.name}" must be at least 1 GiB`);
    }
  });

  return { valid: errors.length === 0, errors };
}

/**
 * Check if a string is a valid CIDR notation
 */
function isValidCIDR(cidr: string): boolean {
  const cidrRegex = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/;
  if (!cidrRegex.test(cidr)) return false;

  const [ip, prefix] = cidr.split('/');
  const prefixNum = parseInt(prefix, 10);
  if (prefixNum < 0 || prefixNum > 32) return false;

  const octets = ip.split('.');
  return octets.every((octet) => {
    const num = parseInt(octet, 10);
    return num >= 0 && num <= 255;
  });
}

/**
 * Get a human-readable description of an action
 */
export function getActionDescription(action: PlanAction): string {
  switch (action.operation) {
    case 'CREATE_VM':
      return `Create VM "${action.params.name}" (${action.params.vcpu_count} vCPUs, ${action.params.memory_mib} MiB)`;
    case 'CREATE_NETWORK':
      return `Create Network "${action.params.name}" (${action.params.cidr || 'auto'})`;
    case 'CREATE_VOLUME':
      return `Create Volume "${action.params.name}" (${action.params.size_gib} GiB)`;
    case 'ATTACH_NETWORK':
      return `Attach network to VM`;
    case 'ATTACH_VOLUME':
      return `Attach volume to VM`;
    default:
      return `${action.operation} ${action.resource_type}`;
  }
}

/**
 * Estimate deployment time based on plan actions
 */
export function estimateDeploymentTime(actions: PlanAction[]): number {
  // Rough estimates in seconds
  const estimates: Record<string, number> = {
    CREATE_VM: 30,
    CREATE_NETWORK: 5,
    CREATE_VOLUME: 10,
    ATTACH_NETWORK: 2,
    ATTACH_VOLUME: 2,
  };

  return actions.reduce((total, action) => {
    return total + (estimates[action.operation] || 10);
  }, 0);
}
