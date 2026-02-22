'use client';

import { useDesignerStore } from '@/store/designer';
import type { DesignerNodeType } from '@/types/designer';
import { Server, Network, Database, X, Trash2 } from 'lucide-react';
import { cn } from '@/lib/utils';

export function PropertyPanel() {
    const {
        selectedNode,
        templateName,
        templateDescription,
        setTemplateName,
        setTemplateDescription,
        updateNodeData,
        removeNode,
        selectNode,
        clearCanvas,
    } = useDesignerStore();

    if (!selectedNode) {
        return (
            <div className="flex h-full flex-col border-l bg-white">
                <div className="border-b p-4">
                    <h2 className="text-sm font-semibold text-slate-900">
                        Template Properties
                    </h2>
                </div>
                <div className="flex-1 space-y-4 p-4">
                    <div>
                        <label className="block text-xs font-medium text-slate-700">
                            Name
                        </label>
                        <input
                            type="text"
                            value={templateName}
                            onChange={(e) => setTemplateName(e.target.value)}
                            className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            placeholder="Template name"
                        />
                    </div>
                    <div>
                        <label className="block text-xs font-medium text-slate-700">
                            Description
                        </label>
                        <textarea
                            value={templateDescription}
                            onChange={(e) =>
                                setTemplateDescription(e.target.value)
                            }
                            rows={3}
                            className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            placeholder="Template description"
                        />
                    </div>
                </div>
                <div className="border-t p-4">
                    <button
                        onClick={clearCanvas}
                        className="flex w-full items-center justify-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-100"
                    >
                        <Trash2 className="h-4 w-4" />
                        Clear Canvas
                    </button>
                </div>
            </div>
        );
    }

    const nodeData = selectedNode.data;
    const isVM = selectedNode.type === 'vm';
    const isNetwork = selectedNode.type === 'network' || selectedNode.type === 'bridge' || selectedNode.type === 'vxlan' || selectedNode.type === 'tap';
    const isVolume = selectedNode.type === 'volume';

    return (
        <div className="flex h-full flex-col border-l bg-white">
            <div className="flex items-center justify-between border-b p-4">
                <div className="flex items-center gap-2">
                    {isVM && (
                        <div className="flex h-8 w-8 items-center justify-center rounded bg-indigo-100 text-indigo-600">
                            <Server className="h-4 w-4" />
                        </div>
                    )}
                    {isNetwork && (
                        <div className="flex h-8 w-8 items-center justify-center rounded bg-emerald-100 text-emerald-600">
                            <Network className="h-4 w-4" />
                        </div>
                    )}
                    {isVolume && (
                        <div className="flex h-8 w-8 items-center justify-center rounded bg-amber-100 text-amber-600">
                            <Database className="h-4 w-4" />
                        </div>
                    )}
                    <div>
                        <h2 className="text-sm font-semibold text-slate-900">
                            {nodeData.label}
                        </h2>
                        <p className="text-xs text-slate-500 capitalize">
                            {nodeData.type} Properties
                        </p>
                    </div>
                </div>
                <button
                    onClick={() => selectNode(null)}
                    className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
                >
                    <X className="h-4 w-4" />
                </button>
            </div>

            <div className="flex-1 space-y-4 overflow-y-auto p-4">
                {/* Common Properties */}
                <div>
                    <label className="block text-xs font-medium text-slate-700">
                        Label
                    </label>
                    <input
                        type="text"
                        value={nodeData.label}
                        onChange={(e) =>
                            updateNodeData(selectedNode.id, { label: e.target.value })
                        }
                        className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                </div>

                {/* VM Properties */}
                {isVM && (
                    <>
                        <div className="grid grid-cols-2 gap-3">
                            <div>
                                <label className="block text-xs font-medium text-slate-700">
                                    vCPUs
                                </label>
                                <input
                                    type="number"
                                    value={(nodeData as any).vcpuCount || 2}
                                    onChange={(e) =>
                                        updateNodeData(selectedNode.id, {
                                            vcpuCount: parseInt(e.target.value) || 1,
                                        })
                                    }
                                    min={1}
                                    max={64}
                                    className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                                />
                            </div>
                            <div>
                                <label className="block text-xs font-medium text-slate-700">
                                    Memory (MiB)
                                </label>
                                <input
                                    type="number"
                                    value={(nodeData as any).memoryMib || 512}
                                    onChange={(e) =>
                                        updateNodeData(selectedNode.id, {
                                            memoryMib: parseInt(e.target.value) || 512,
                                        })
                                    }
                                    min={512}
                                    step={512}
                                    className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                                />
                            </div>
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Kernel
                            </label>
                            <input
                                type="text"
                                value={(nodeData as any).kernel || ''}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        kernel: e.target.value,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                                placeholder="e.g., /boot/vmlinux"
                            />
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Root FS
                            </label>
                            <input
                                type="text"
                                value={(nodeData as any).rootfs || ''}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        rootfs: e.target.value,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                                placeholder="e.g., /path/to/rootfs.ext4"
                            />
                        </div>
                    </>
                )}

                {/* Network Properties */}
                {isNetwork && (
                    <>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Network Name
                            </label>
                            <input
                                type="text"
                                value={(nodeData as any).name || ''}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        name: e.target.value,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            />
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Type
                            </label>
                            <select
                                value={(nodeData as any).networkType || 'bridge'}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        networkType: e.target.value as any,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            >
                                <option value="bridge">Bridge</option>
                                <option value="vxlan">VXLAN</option>
                                <option value="tap">TAP</option>
                            </select>
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                CIDR (optional)
                            </label>
                            <input
                                type="text"
                                value={(nodeData as any).cidr || ''}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        cidr: e.target.value || undefined,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                                placeholder="e.g., 192.168.1.0/24"
                            />
                        </div>
                    </>
                )}

                {/* Volume Properties */}
                {isVolume && (
                    <>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Volume Name
                            </label>
                            <input
                                type="text"
                                value={(nodeData as any).name || ''}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        name: e.target.value,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            />
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Size (GB)
                            </label>
                            <input
                                type="number"
                                value={(nodeData as any).sizeGb || 10}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        sizeGb: parseInt(e.target.value) || 10,
                                    })
                                }
                                min={1}
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            />
                        </div>
                        <div>
                            <label className="block text-xs font-medium text-slate-700">
                                Format
                            </label>
                            <select
                                value={(nodeData as any).format || 'raw'}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        format: e.target.value as any,
                                    })
                                }
                                className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            >
                                <option value="raw">Raw</option>
                                <option value="qcow2">QCOW2</option>
                            </select>
                        </div>
                        <div className="flex items-center gap-2">
                            <input
                                type="checkbox"
                                id="isPersistent"
                                checked={(nodeData as any).isPersistent ?? true}
                                onChange={(e) =>
                                    updateNodeData(selectedNode.id, {
                                        isPersistent: e.target.checked,
                                    })
                                }
                                className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                            />
                            <label
                                htmlFor="isPersistent"
                                className="text-sm text-slate-700"
                            >
                                Persistent
                            </label>
                        </div>
                    </>
                )}
            </div>

            <div className="border-t p-4">
                <button
                    onClick={() => {
                        removeNode(selectedNode.id);
                    }}
                    className="flex w-full items-center justify-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-100"
                >
                    <Trash2 className="h-4 w-4" />
                    Delete Node
                </button>
            </div>
        </div>
    );
}
