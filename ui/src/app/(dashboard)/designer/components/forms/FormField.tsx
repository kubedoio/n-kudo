'use client';

import { ReactNode } from 'react';

interface FormFieldProps {
  label: string;
  children: ReactNode;
  className?: string;
}

export function FormField({ label, children, className = '' }: FormFieldProps) {
  return (
    <div className={`space-y-1.5 ${className}`}>
      <label className="block text-xs font-medium text-slate-500 uppercase tracking-wide">
        {label}
      </label>
      {children}
    </div>
  );
}
