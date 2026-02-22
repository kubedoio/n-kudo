'use client';

import { VolumeNodeData } from '../types';
import { FormField, TextInput, NumberInput, CheckboxInput } from './forms';

interface VolumePropertyFormProps {
  data: VolumeNodeData;
  onChange: (data: Partial<VolumeNodeData>) => void;
}

export function VolumePropertyForm({ data, onChange }: VolumePropertyFormProps) {
  return (
    <div className="space-y-4">
      <div className="border-b pb-3">
        <h3 className="text-sm font-semibold text-slate-900">Volume Configuration</h3>
        <p className="text-xs text-slate-500 mt-0.5">Configure storage volume settings</p>
      </div>

      <FormField label="Name">
        <TextInput
          value={data.name}
          onChange={(value) => onChange({ name: value })}
          placeholder="Enter volume name"
        />
      </FormField>

      <FormField label="Size (GiB)">
        <NumberInput
          value={data.sizeGb}
          onChange={(value) => onChange({ sizeGb: value })}
          min={1}
          step={1}
        />
      </FormField>

      <div className="pt-2">
        <CheckboxInput
          checked={data.isPersistent}
          onChange={(checked) => onChange({ isPersistent: checked })}
          label="Persistent volume"
        />
        <p className="text-xs text-slate-400 mt-1 ml-6">
          Persistent volumes retain data across VM restarts
        </p>
      </div>
    </div>
  );
}
