'use client';

import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { HardDrive, Database } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { VolumeNodeData } from '@/types/designer';

interface VolumeNodeProps extends NodeProps {
  data: VolumeNodeData;

}

export const VolumeNode = memo(function VolumeNode({ data, selected }: VolumeNodeProps) {
  return (
    <div
      className={cn(
        'w-44 rounded-lg border-2 bg-white shadow-sm transition-all',
        selected
          ? 'border-amber-500 shadow-md ring-2 ring-amber-100'
          : 'border-slate-200 hover:border-amber-300'
      )}
    >
      {/* Header */}
      <div className="flex items-center gap-2 rounded-t-lg bg-amber-50 px-3 py-2">
        <div className="flex h-6 w-6 items-center justify-center rounded bg-amber-100">
          <Database className="h-3.5 w-3.5 text-amber-600" />
        </div>
        <span className="flex-1 truncate text-sm font-semibold text-amber-900">
          {data.name || data.label}
        </span>
      </div>

      {/* Body */}
      <div className="space-y-2 p-3">
        <div className="flex items-center gap-2 text-xs text-slate-600">
          <HardDrive className="h-3.5 w-3.5 text-slate-400" />
          <span>{data.sizeGb} GB</span>
        </div>
        <div className="text-xs text-slate-500 uppercase">
          {data.format}
        </div>
        <div className="text-xs text-slate-400">
          {data.isPersistent ? 'Persistent' : 'Ephemeral'}
        </div>
      </div>

      {/* Handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="!h-3 !w-3 !border-2 !border-white !bg-amber-500"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="!h-3 !w-3 !border-2 !border-white !bg-amber-500"
      />
    </div>
  );
});
