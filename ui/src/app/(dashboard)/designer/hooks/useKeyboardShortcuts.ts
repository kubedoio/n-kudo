'use client';

import { useEffect, useCallback } from 'react';

interface KeyboardShortcutHandlers {
  onDelete: () => void;
  onSave: () => void;
  onUndo?: () => void;
}

export function useKeyboardShortcuts(handlers: KeyboardShortcutHandlers) {
  const { onDelete, onSave, onUndo } = handlers;

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't trigger shortcuts when typing in input fields
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLSelectElement
      ) {
        return;
      }

      // Delete key - delete selected node
      if (e.key === 'Delete' || e.key === 'Backspace') {
        e.preventDefault();
        onDelete();
      }

      // Ctrl/Cmd + S - save template
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        onSave();
      }

      // Ctrl/Cmd + Z - undo (if handler provided)
      if ((e.metaKey || e.ctrlKey) && e.key === 'z' && onUndo) {
        e.preventDefault();
        onUndo();
      }
    },
    [onDelete, onSave, onUndo]
  );

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);
}
