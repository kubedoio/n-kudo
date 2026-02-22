'use client';

interface TextInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

export function TextInput({
  value,
  onChange,
  placeholder,
  disabled = false,
  className = '',
}: TextInputProps) {
  return (
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      className={`
        block w-full rounded-lg border border-slate-200 px-3 py-2 text-sm
        text-slate-900 placeholder:text-slate-400
        focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500
        disabled:bg-slate-50 disabled:text-slate-500
        ${className}
      `}
    />
  );
}
