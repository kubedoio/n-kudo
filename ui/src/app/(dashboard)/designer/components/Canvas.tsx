'use client';

import { ReactFlow, Background, Controls, MiniMap, BackgroundVariant } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { nodeTypes } from './nodes';
import { cn } from '@/lib/utils';

interface CanvasProps {
  nodes: any[];
  edges: any[];
  onNodesChange: (nodes: any[]) => void;
  onEdgesChange: (edges: any[]) => void;
  onConnect: (connection: any) => void;
  onNodeClick: (event: any, node: any) => void;
  className?: string;
}

export function Canvas({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onConnect,
  onNodeClick,
  className,
}: CanvasProps) {
  return (
    <div className={cn('flex-1 h-full', className)}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-left"
        snapToGrid
        snapGrid={[15, 15]}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color="#cbd5e1"
        />
        <Controls className="bg-white shadow-md" />
        <MiniMap
          className="bg-white shadow-md"
          nodeStrokeWidth={3}
          nodeStrokeColor="#6366f1"
          maskColor="rgba(241, 245, 249, 0.7)"
        />
      </ReactFlow>
    </div>
  );
}
