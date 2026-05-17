import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Button, IconButton, InlineField, InlineFieldRow, Input, Select } from '@grafana/ui';

import { Calculation, CalculationOp, CALCULATION_OPS, COLUMN_REQUIRED_OPS } from '../../types';

interface Props {
  calculations: Calculation[];
  columnOptions: Array<SelectableValue<string>>;
  loadingColumns: boolean;
  onChange: (calculations: Calculation[]) => void;
}

/**
 * CalculationsEditor renders a list of calculation rows (op + optional column + optional alias).
 * Users can add/remove calculations and reorder them.
 */
export function CalculationsEditor({ calculations, columnOptions, loadingColumns, onChange }: Props) {
  const add = () => {
    onChange([...calculations, { op: 'COUNT' }]);
  };

  const remove = (idx: number) => {
    onChange(calculations.filter((_, i) => i !== idx));
  };

  const updateCalc = (idx: number, partial: Partial<Calculation>) => {
    const next = calculations.map((c, i) => (i === idx ? { ...c, ...partial } : c));
    onChange(next);
  };

  return (
    <div>
      {calculations.map((calc, idx) => {
        const needsColumn = COLUMN_REQUIRED_OPS.has(calc.op as CalculationOp);
        return (
          <InlineFieldRow key={idx}>
            <InlineField label={idx === 0 ? 'Operation' : ''} labelWidth={10}>
              <Select
                options={CALCULATION_OPS}
                value={calc.op}
                onChange={(v) => {
                  const op = (v.value ?? 'COUNT') as CalculationOp;
                  const update: Partial<Calculation> = { op };
                  // Clear column if the new op doesn't need one.
                  if (!COLUMN_REQUIRED_OPS.has(op)) {
                    update.column = undefined;
                  }
                  updateCalc(idx, update);
                }}
                width={16}
              />
            </InlineField>

            {needsColumn && (
              <InlineField label={idx === 0 ? 'Column' : ''} labelWidth={8}>
                <Select
                  options={columnOptions}
                  value={calc.column}
                  onChange={(v) => updateCalc(idx, { column: v.value })}
                  allowCustomValue
                  placeholder="column name"
                  isLoading={loadingColumns}
                  width={20}
                />
              </InlineField>
            )}

            <InlineField label={idx === 0 ? 'Alias' : ''} labelWidth={6} tooltip="Optional display name for this metric">
              <Input
                placeholder="optional alias"
                value={calc.alias || ''}
                onChange={(e) => updateCalc(idx, { alias: e.currentTarget.value || undefined })}
                width={16}
              />
            </InlineField>

            <IconButton
              name="trash-alt"
              tooltip="Remove calculation"
              onClick={() => remove(idx)}
              disabled={calculations.length === 1}
            />
          </InlineFieldRow>
        );
      })}

      <Button
        variant="secondary"
        size="sm"
        icon="plus"
        onClick={add}
        disabled={calculations.length >= 100}
      >
        Add calculation
      </Button>
    </div>
  );
}
