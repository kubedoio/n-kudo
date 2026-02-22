'use client';

import { Box, Network, HardDrive, MousePointer2, Sparkles } from 'lucide-react';
import { cn } from '@/lib/utils';

interface EmptyStateProps {
  onAddNode: (type: 'vm' | 'network' | 'volume') => void;
}

const quickStartItems = [
  {
    type: 'vm' as const,
    label: 'Add VM',
    description: 'Virtual Machine',
    icon: Box,
    color: 'indigo',
  },
  {
    type: 'network' as const,
    label: 'Add Network',
    description: 'Virtual Network',
    icon: Network,
    color: 'emerald',
  },
  {
    type: 'volume' as const,
    label: 'Add Volume',
    description: 'Storage Volume',
    icon: HardDrive,
    color: 'amber',
  },
];

const colorClasses: Record<string, { bg: string; icon: string; border: string; hover: string }> = {
  indigo: {
    bg: 'bg-indigo-50',
    icon: 'text-indigo-600',
    border: 'border-indigo-200',
    hover: 'hover:border-indigo-300 hover:bg-indigo-100',
  },
  emerald: {
    bg: 'bg-emerald-50',
    icon: 'text-emerald-600',
    border: 'border-emerald-200',
    hover: 'hover:border-emerald-300 hover:bg-emerald-100',
  },
  amber: {
    bg: 'bg-amber-50',
    icon: 'text-amber-600',
    border: 'border-amber-200',
    hover: 'hover:border-amber-300 hover:bg-amber-100',
  },
};

export function EmptyState({ onAddNode }: EmptyStateProps) {
  return (
    <div className="flex h-full flex-col items-center justify-center p-8">
      {/* Welcome Message */}
      <div className="mb-8 text-center">
        <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-indigo-100">
          <Sparkles className="h-8 w-8 text-indigo-600" />
        </div>
        <h3 className="mb-2 text-xl font-semibold text-slate-900">
          Welcome to the Infrastructure Designer
        </h3>
        <p className="max-w-md text-sm text-slate-500">
          Design your infrastructure by adding VMs, networks, and storage volumes. 
          Connect them together to create a complete deployment template.
        </p>
      </div>

      {/* Quick Start Buttons */}
      <div className="mb-8 grid grid-cols-3 gap-4">
        {quickStartItems.map((item) => {
          const colors = colorClasses[item.color];
          const Icon = item.icon;
          return (
            <button
              key={item.type}
              onClick={() => onAddNode(item.type)}
              className={cn(
                'flex flex-col items-center gap-3 rounded-xl border-2 bg-white p-5 transition-all',
                'border-slate-100 hover:shadow-md',
                colors.hover
              )}
            >
              <div className={cn('flex h-12 w-12 items-center justify-center rounded-xl', colors.bg)}>
                <Icon className={cn('h-6 w-6', colors.icon)} />
              </div>
              <div className="text-center">
                <p className="text-sm font-medium text-slate-900">{item.label}</p>
                <p className="text-xs text-slate-500">{item.description}</p>
              </div>
            </button>
          );
        })}
      </div>

      {/* Hint */}
      <div className="flex items-center gap-2 rounded-lg bg-slate-50 px-4 py-3 text-sm text-slate-500">
        <MousePointer2 className="h-4 w-4" />
        <span>
          You can also drag nodes from the palette on the left, or use keyboard shortcuts:
          <span className="ml-1 font-medium text-slate-700">Delete</span> to remove,
          <span className="ml-1 font-medium text-slate-700">Ctrl+S</span> to save
        </span>
      </div>
    </div>
  );
}
