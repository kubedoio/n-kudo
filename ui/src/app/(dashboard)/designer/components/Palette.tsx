'use client';

import { useCallback } from 'react';
import { Box, Network, HardDrive } from 'lucide-react';
import { cn } from '@/lib/utils';

type NodeType = 'vm' | 'network' | 'volume';

interface PaletteItem {
  type: NodeType;
  label: string;
  description: string;
  icon: React.ReactNode;
  category: 'compute' | 'networking' | 'storage';
  color: 'indigo' | 'emerald' | 'amber';
}

const paletteItems: PaletteItem[] = [
  {
    type: 'vm',
    label: 'VM Node',
    description: 'Virtual Machine',
    icon: <Box className="h-5 w-5" />,
    category: 'compute',
    color: 'indigo',
  },
  {
    type: 'network',
    label: 'Network Node',
    description: 'Virtual Network',
    icon: <Network className="h-5 w-5" />,
    category: 'networking',
    color: 'emerald',
  },
  {
    type: 'volume',
    label: 'Volume Node',
    description: 'Storage Volume',
    icon: <HardDrive className="h-5 w-5" />,
    category: 'storage',
    color: 'amber',
  },
];

const categoryLabels: Record<string, string> = {
  compute: 'Compute',
  networking: 'Networking',
  storage: 'Storage',
};

const colorClasses: Record<string, { bg: string; text: string; border: string }> = {
  indigo: {
    bg: 'bg-indigo-50',
    text: 'text-indigo-600',
    border: 'border-indigo-200',
  },
  emerald: {
    bg: 'bg-emerald-50',
    text: 'text-emerald-600',
    border: 'border-emerald-200',
  },
  amber: {
    bg: 'bg-amber-50',
    text: 'text-amber-600',
    border: 'border-amber-200',
  },
};

export function Palette() {
  const onDragStart = useCallback(
    (event: React.DragEvent<HTMLDivElement>, nodeType: NodeType) => {
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
    <div className="flex h-full w-64 flex-col border-r border-slate-200 bg-slate-50">
      {/* Header */}
      <div className="border-b border-slate-200 bg-white px-4 py-3">
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
                {items.map((item) => {
                  const colors = colorClasses[item.color];
                  return (
                    <div
                      key={item.type}
                      className={cn(
                        'group cursor-grab rounded-xl border bg-white p-3 shadow-sm transition-all hover:shadow-md active:cursor-grabbing',
                        'border-transparent hover:border-slate-300'
                      )}
                      draggable
                      onDragStart={(e) => onDragStart(e, item.type)}
                    >
                      <div className="flex items-start gap-3">
                        <div
                          className={cn(
                            'flex h-9 w-9 shrink-0 items-center justify-center rounded-lg',
                            colors.bg,
                            colors.text
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
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>

      {/* Footer hint */}
      <div className="border-t border-slate-200 bg-slate-100 px-4 py-3">
        <p className="text-[10px] text-slate-500">
          Drag items onto the canvas to add them to your infrastructure design.
        </p>
      </div>
    </div>
  );
}
