'use client';

import { Minus, Plus } from 'lucide-react';

interface NumberInputProps {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  disabled?: boolean;
  className?: string;
}

export function NumberInput({
  value,
  onChange,
  min,
  max,
  step = 1,
  disabled = false,
  className = '',
}: NumberInputProps) {
  const handleDecrement = () => {
    if (disabled) return;
    const newValue = value - step;
    if (min !== undefined && newValue < min) return;
    onChange(newValue);
  };

  const handleIncrement = () => {
    if (disabled) return;
    const newValue = value + step;
    if (max !== undefined && newValue > max) return;
    onChange(newValue);
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = parseFloat(e.target.value);
    if (isNaN(newValue)) return;
    if (min !== undefined && newValue < min) return;
    if (max !== undefined && newValue > max) return;
    onChange(newValue);
  };

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <button
        type="button"
        onClick={handleDecrement}
        disabled={disabled || (min !== undefined && value <= min)}
        className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 hover:text-slate-900 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <Minus className="h-4 w-4" />
      </button>
      <input
        type="number"
        value={value}
        onChange={handleChange}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        className="
          block w-full rounded-lg border border-slate-200 px-3 py-2 text-sm text-center
          text-slate-900
          focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500
          disabled:bg-slate-50 disabled:text-slate-500
        "
      />
      <button
        type="button"
        onClick={handleIncrement}
        disabled={disabled || (max !== undefined && value >= max)}
        className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 hover:text-slate-900 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <Plus className="h-4 w-4" />
      </button>
    </div>
  );
}
