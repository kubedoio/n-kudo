'use client';

interface CheckboxInputProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
  className?: string;
}

export function CheckboxInput({
  checked,
  onChange,
  label,
  disabled = false,
  className = '',
}: CheckboxInputProps) {
  const checkbox = (
    <input
      type="checkbox"
      checked={checked}
      onChange={(e) => onChange(e.target.checked)}
      disabled={disabled}
      className="
        h-4 w-4 rounded border-slate-300 text-indigo-600
        focus:ring-indigo-500 focus:ring-offset-0
        disabled:opacity-50 disabled:cursor-not-allowed
      "
    />
  );

  if (label) {
    return (
      <label className={`flex items-center gap-2 cursor-pointer ${className}`}>
        {checkbox}
        <span className="text-sm text-slate-700">{label}</span>
      </label>
    );
  }

  return <div className={className}>{checkbox}</div>;
}
