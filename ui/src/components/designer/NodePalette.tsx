'use client';

import { useCallback } from 'react';
import {
  Cpu,
  Network,
  HardDrive,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { DesignerNodeType } from '@/types/designer';

// For MVP-2 we map 'network' type to available network node types
type PaletteNodeType = DesignerNodeType;

interface PaletteItem {
  type: DesignerNodeType;
  label: string;
  description: string;
  icon: React.ReactNode;
  category: 'compute' | 'networking' | 'storage';
  color: string;
}

const paletteItems: PaletteItem[] = [
  {
    type: 'vm',
    label: 'Virtual Machine',
    description: 'Cloud Hypervisor VM',
    icon: <Cpu className="h-5 w-5" />,
    category: 'compute',
    color: 'indigo',
  },
  {
    type: 'network',
    label: 'Network',
    description: 'Virtual network (Bridge/VXLAN/TAP)',
    icon: <Network className="h-5 w-5" />,
    category: 'networking',
    color: 'emerald',
  },
  {
    type: 'volume',
    label: 'Storage Volume',
    description: 'Block storage for VMs',
    icon: <HardDrive className="h-5 w-5" />,
    category: 'storage',
    color: 'blue',
  },
];

const categoryLabels: Record<string, string> = {
  compute: 'Compute',
  networking: 'Networking',
  storage: 'Storage',
};

export function NodePalette() {
  const onDragStart = useCallback(
    (event: React.DragEvent<HTMLDivElement>, nodeType: DesignerNodeType) => {
      event.dataTransfer.setData('application/reactflow', nodeType);
      event.dataTransfer.effectAllowed = 'move';
    },
    []
  );

  // Group items by category
  const groupedItems = paletteItems.reduce((acc, item) => {
    if (!acc[item.category]) {
      acc[item.category] = [];
    }
    acc[item.category].push(item);
    return acc;
  }, {} as Record<string, PaletteItem[]>);

  const categories = ['compute', 'networking', 'storage'] as const;

  return (
    <div className="flex h-full w-60 flex-col border-r bg-slate-50">
      {/* Header */}
      <div className="border-b bg-white px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-900">Node Palette</h2>
        <p className="text-xs text-slate-500">Drag to add nodes</p>
      </div>

      {/* Items */}
      <div className="flex-1 overflow-y-auto py-2">
        {categories.map((category) => {
          const items = groupedItems[category];
          if (!items?.length) return null;

          return (
            <div key={category} className="mb-4">
              {/* Category Header */}
              <div className="px-4 py-2">
                <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                  {categoryLabels[category]}
                </span>
              </div>

              {/* Category Items */}
              <div className="space-y-1 px-2">
                {items.map((item) => (
                  <div
                    key={item.type}
                    className={cn(
                      'group cursor-grab rounded-lg border border-transparent bg-white p-3 shadow-sm transition-all hover:border-slate-300 hover:shadow-md active:cursor-grabbing'
                    )}
                    draggable
                    onDragStart={(e) => onDragStart(e, item.type)}
                  >
                    <div className="flex items-start gap-3">
                      <div
                        className={cn(
                          'flex h-8 w-8 shrink-0 items-center justify-center rounded-md',
                          item.color === 'indigo' && 'bg-indigo-50 text-indigo-600',
                          item.color === 'emerald' && 'bg-emerald-50 text-emerald-600',
                          item.color === 'blue' && 'bg-blue-50 text-blue-600'
                        )}
                      >
                        {item.icon}
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium text-slate-900">
                          {item.label}
                        </p>
                        <p className="truncate text-xs text-slate-500">
                          {item.description}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {/* Footer hint */}
      <div className="border-t bg-slate-100 px-4 py-3">
        <p className="text-[10px] text-slate-500">
          Drag items onto the canvas to add them to your infrastructure design.
        </p>
      </div>
    </div>
  );
}
