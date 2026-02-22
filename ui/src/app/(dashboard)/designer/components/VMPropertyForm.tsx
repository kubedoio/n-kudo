'use client';

import { VMNodeData } from '../types';
import { FormField, TextInput, NumberInput, SelectInput } from './forms';

interface VMPropertyFormProps {
  data: VMNodeData;
  onChange: (data: Partial<VMNodeData>) => void;
}

const IMAGE_OPTIONS = [
  { value: 'ubuntu-22.04', label: 'Ubuntu 22.04 LTS' },
  { value: 'debian-12', label: 'Debian 12 (Bookworm)' },
  { value: 'alpine-3.19', label: 'Alpine Linux 3.19' },
];

export function VMPropertyForm({ data, onChange }: VMPropertyFormProps) {
  return (
    <div className="space-y-4">
      <div className="border-b pb-3">
        <h3 className="text-sm font-semibold text-slate-900">VM Configuration</h3>
        <p className="text-xs text-slate-500 mt-0.5">Configure virtual machine resources</p>
      </div>

      <FormField label="Name">
        <TextInput
          value={data.name}
          onChange={(value) => onChange({ name: value })}
          placeholder="Enter VM name"
        />
      </FormField>

      <FormField label="vCPU Count">
        <NumberInput
          value={data.vcpuCount}
          onChange={(value) => onChange({ vcpuCount: value })}
          min={1}
          max={32}
          step={1}
        />
      </FormField>

      <FormField label="Memory (MiB)">
        <NumberInput
          value={data.memoryMib}
          onChange={(value) => onChange({ memoryMib: value })}
          min={256}
          step={256}
        />
      </FormField>

      <FormField label="Disk Size (GiB)">
        <NumberInput
          value={data.diskSizeGb}
          onChange={(value) => onChange({ diskSizeGb: value })}
          min={1}
          step={1}
        />
      </FormField>

      <FormField label="Image">
        <SelectInput
          value={data.image}
          onChange={(value) => onChange({ image: value as VMNodeData['image'] })}
          options={IMAGE_OPTIONS}
        />
      </FormField>
    </div>
  );
}
