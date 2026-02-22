'use client';

interface SelectOption {
  value: string;
  label: string;
}

interface SelectInputProps {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  disabled?: boolean;
  className?: string;
}

export function SelectInput({
  value,
  onChange,
  options,
  disabled = false,
  className = '',
}: SelectInputProps) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className={`
        block w-full rounded-lg border border-slate-200 px-3 py-2 text-sm
        text-slate-900
        focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500
        disabled:bg-slate-50 disabled:text-slate-500
        bg-white
        ${className}
      `}
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  );
}
