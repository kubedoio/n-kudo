'use client';

import { useCallback, useRef, useEffect } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  Edge,
  Node,
  ReactFlowProvider,
  useReactFlow,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { nodeTypes } from './nodes';
import { useDesignerStore } from '@/store/designer';
import type { DesignerNodeType, DesignerNode } from '@/types/designer';
import { cn } from '@/lib/utils';

// Transform store nodes to React Flow nodes
const toReactFlowNodes = (nodes: DesignerNode[]): Node[] => {
  return nodes.map((node) => ({
    id: node.id,
    type: node.type,
    position: node.position,
    data: node.data,
  }));
};

function CanvasContent() {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const { screenToFlowPosition } = useReactFlow();
  
  // Store access
  const store = useDesignerStore();
  
  // Local React Flow state
  const [nodes, setNodes, onNodesChange] = useNodesState<any>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<any>([]);

  // Sync store state to React Flow
  useEffect(() => {
    setNodes(toReactFlowNodes(store.nodes));
    setEdges(store.edges.map((e: any) => ({ ...e, type: 'smoothstep', animated: true })));
  }, [store.nodes, store.edges, setNodes, setEdges]);

  // Handle connections (edge creation)
  const onConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return;
      
      const newEdge = {
        id: `e-${connection.source}-${connection.target}`,
        source: connection.source,
        target: connection.target,
        sourceHandle: connection.sourceHandle || undefined,
        targetHandle: connection.targetHandle || undefined,
      };
      
      setEdges((eds) => addEdge({ ...newEdge, type: 'smoothstep', animated: true }, eds));
      store.onConnect(connection);
    },
    [setEdges, store]
  );

  // Handle node selection
  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      const storeNode = store.nodes.find((n) => n.id === node.id);
      if (storeNode) {
        store.selectNode(storeNode);
      }
    },
    [store]
  );

  // Handle background click (deselect)
  const onPaneClick = useCallback(() => {
    store.selectNode(null);
  }, [store]);

  // Handle drag over for drop
  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  // Handle drop from palette
  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const type = event.dataTransfer.getData('application/reactflow') as DesignerNodeType;
      
      if (!type || !reactFlowWrapper.current) {
        return;
      }

      // Calculate drop position
      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      // Create new node via store
      store.addNode({ type, position });
    },
    [screenToFlowPosition, store]
  );

  // Handle node deletion via keypress
  const onKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if ((event.key === 'Delete' || event.key === 'Backspace') && store.selectedNodeId) {
        store.removeNode(store.selectedNodeId);
      }
    },
    [store]
  );

  return (
    <div
      ref={reactFlowWrapper}
      className="h-full w-full"
      onKeyDown={onKeyDown}
      tabIndex={0}
    >
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        onDragOver={onDragOver}
        onDrop={onDrop}
        nodeTypes={nodeTypes}
        fitView
        attributionPosition="bottom-left"
        deleteKeyCode={['Delete', 'Backspace']}
        selectionKeyCode="Shift"
        multiSelectionKeyCode="Control"
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

interface DesignerCanvasProps {
  className?: string;
}

export function DesignerCanvas({ className }: DesignerCanvasProps) {
  return (
    <div className={cn('relative h-full w-full bg-slate-50', className)}>
      <ReactFlowProvider>
        <CanvasContent />
      </ReactFlowProvider>
    </div>
  );
}
