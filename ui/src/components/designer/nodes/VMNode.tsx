'use client';

import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Cpu, MemoryStick, HardDrive } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { VMNodeData } from '@/types/designer';

interface VMNodeProps extends NodeProps {
  data: VMNodeData;

}

export const VMNode = memo(function VMNode({ data, selected }: VMNodeProps) {
  return (
    <div
      className={cn(
        'w-48 rounded-lg border-2 bg-white shadow-sm transition-all',
        selected
          ? 'border-indigo-500 shadow-md ring-2 ring-indigo-100'
          : 'border-slate-200 hover:border-indigo-300'
      )}
    >
      {/* Header */}
      <div className="flex items-center gap-2 rounded-t-lg bg-indigo-50 px-3 py-2">
        <div className="flex h-6 w-6 items-center justify-center rounded bg-indigo-100">
          <Cpu className="h-3.5 w-3.5 text-indigo-600" />
        </div>
        <span className="flex-1 truncate text-sm font-semibold text-indigo-900">
          {data.name || data.label}
        </span>
      </div>

      {/* Body */}
      <div className="space-y-2 p-3">
        <div className="flex items-center gap-2 text-xs text-slate-600">
          <Cpu className="h-3.5 w-3.5 text-slate-400" />
          <span>{data.vcpuCount || 2} vCPU</span>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-600">
          <MemoryStick className="h-3.5 w-3.5 text-slate-400" />
          <span>{data.memoryMib || 1024} MiB</span>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-600">
          <HardDrive className="h-3.5 w-3.5 text-slate-400" />
          <span className="truncate">{data.rootfs?.split('/').pop() || 'rootfs'}</span>
        </div>
      </div>

      {/* Handles */}
      <Handle
        type="target"
        position={Position.Left}
        className="!h-3 !w-3 !border-2 !border-white !bg-indigo-500"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="!h-3 !w-3 !border-2 !border-white !bg-indigo-500"
      />
    </div>
  );
});
