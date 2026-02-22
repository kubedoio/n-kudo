'use client';

import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Network, Globe } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { NetworkNodeData } from '@/types/designer';

interface NetworkNodeProps extends NodeProps {
  data: NetworkNodeData;
}

export const NetworkNode = memo(function NetworkNode({ data, selected }: NetworkNodeProps) {
  const typeLabels: Record<string, string> = {
    bridge: 'Bridge',
    vxlan: 'VXLAN',
    tap: 'TAP',
  };

  return (
    <div
      className={cn(
        'w-44 rounded-lg border-2 bg-white shadow-sm transition-all',
        selected
          ? 'border-emerald-500 shadow-md ring-2 ring-emerald-100'
          : 'border-slate-200 hover:border-emerald-300'
      )}
    >
      {/* Header */}
      <div className="flex items-center gap-2 rounded-t-lg bg-emerald-50 px-3 py-2">
        <div className="flex h-6 w-6 items-center justify-center rounded bg-emerald-100">
          <Network className="h-3.5 w-3.5 text-emerald-600" />
        </div>
        <span className="flex-1 truncate text-sm font-semibold text-emerald-900">
          {data.name || data.label}
        </span>
      </div>

      {/* Body */}
      <div className="space-y-2 p-3">
        <div className="flex items-center gap-2 text-xs text-slate-600">
          <Globe className="h-3.5 w-3.5 text-slate-400" />
          <span className="font-medium">{typeLabels[data.networkType] || data.networkType}</span>
        </div>
        {data.cidr && (
          <div className="text-xs text-slate-500 font-mono">
            {data.cidr}
          </div>
        )}
      </div>

      {/* Handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="!h-3 !w-3 !border-2 !border-white !bg-emerald-500"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="!h-3 !w-3 !border-2 !border-white !bg-emerald-500"
      />
    </div>
  );
});
