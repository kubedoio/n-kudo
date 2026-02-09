import { useState, useRef, useEffect } from 'react';
import { Play, Square, Trash2, ChevronDown } from 'lucide-react';
import { Button, ConfirmModal } from '@/components/common';
import type { MicroVM } from '@/api/types';

interface VMActionsMenuProps {
  vm: MicroVM;
  onStart: () => void;
  onStop: () => void;
  onDelete: () => void;
  isLoading?: boolean;
}

export function VMActionsMenu({
  vm,
  onStart,
  onStop,
  onDelete,
  isLoading = false,
}: VMActionsMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleAction = (action: 'start' | 'stop' | 'delete') => {
    setIsOpen(false);

    switch (action) {
      case 'start':
        onStart();
        break;
      case 'stop':
        onStop();
        break;
      case 'delete':
        setShowDeleteConfirm(true);
        break;
    }
  };

  const handleConfirmDelete = () => {
    onDelete();
    setShowDeleteConfirm(false);
  };

  const isRunning = vm.state === 'RUNNING';
  const isStopped = vm.state === 'STOPPED';
  const isPending = ['CREATING', 'STARTING', 'STOPPING', 'DELETING'].includes(vm.state);

  return (
    <>
      <div className="relative" ref={menuRef}>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsOpen(!isOpen)}
          disabled={isLoading || isPending}
          rightIcon={<ChevronDown className="w-3 h-3" />}
        >
          Actions
        </Button>

        {isOpen && (
          <div className="absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 z-50">
            <div className="py-1" role="menu">
              {/* Start Action */}
              <button
                onClick={() => handleAction('start')}
                disabled={!isStopped || isLoading}
                className={`
                  w-full text-left px-4 py-2 text-sm flex items-center gap-2
                  ${isStopped
                    ? 'text-gray-700 hover:bg-gray-100'
                    : 'text-gray-400 cursor-not-allowed'
                  }
                `}
                role="menuitem"
              >
                <Play className="w-4 h-4" />
                Start VM
              </button>

              {/* Stop Action */}
              <button
                onClick={() => handleAction('stop')}
                disabled={!isRunning || isLoading}
                className={`
                  w-full text-left px-4 py-2 text-sm flex items-center gap-2
                  ${isRunning
                    ? 'text-gray-700 hover:bg-gray-100'
                    : 'text-gray-400 cursor-not-allowed'
                  }
                `}
                role="menuitem"
              >
                <Square className="w-4 h-4" />
                Stop VM
              </button>

              {/* Divider */}
              <div className="border-t border-gray-100 my-1" />

              {/* Delete Action */}
              <button
                onClick={() => handleAction('delete')}
                disabled={isLoading || isPending}
                className={`
                  w-full text-left px-4 py-2 text-sm flex items-center gap-2
                  ${!isLoading && !isPending
                    ? 'text-red-600 hover:bg-red-50'
                    : 'text-gray-400 cursor-not-allowed'
                  }
                `}
                role="menuitem"
              >
                <Trash2 className="w-4 h-4" />
                Delete VM
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleConfirmDelete}
        title="Delete Virtual Machine"
        confirmLabel="Delete"
        confirmVariant="danger"
        isLoading={isLoading}
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            Are you sure you want to delete the VM <strong>{vm.name}</strong>? This action
            cannot be undone.
          </p>
          
          <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3">
            <div className="flex items-start gap-2">
              <svg
                className="w-5 h-5 text-yellow-600 mt-0.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
              <div className="text-sm text-yellow-800">
                <p className="font-medium">Warning</p>
                <p className="mt-1">
                  All data on this VM will be permanently deleted. Make sure you have
                  backed up any important data before proceeding.
                </p>
              </div>
            </div>
          </div>

          <div className="text-sm text-gray-500">
            <p>VM Details:</p>
            <ul className="mt-1 ml-4 list-disc">
              <li>Name: {vm.name}</li>
              <li>ID: {vm.id}</li>
              <li>vCPUs: {vm.vcpu_count}</li>
              <li>Memory: {vm.memory_mib} MiB</li>
            </ul>
          </div>
        </div>
      </ConfirmModal>
    </>
  );
}
