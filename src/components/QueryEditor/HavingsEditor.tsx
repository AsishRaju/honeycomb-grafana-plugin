import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Button, IconButton, InlineField, InlineFieldRow, Input, Select } from '@grafana/ui';

import {
  Calculation,
  CalculationOp,
  COLUMN_REQUIRED_OPS,
  Having,
  HAVING_OPS,
  HavingOp,
} from '../../types';

interface Props {
  havings: Having[];
  calculations: Calculation[];
  onChange: (havings: Having[]) => void;
}

/**
 * HavingsEditor renders a list of post-aggregation filter rows.
 *
 * Each row references one of the query's calculations (by op + optional column)
 * and applies a comparison (e.g. "P95(duration_ms) > 500"). The Calculation
 * dropdown is populated from the current query's calculations to keep the
 * having tied to a real aggregated value.
 */
export function HavingsEditor({ havings, calculations, onChange }: Props) {
  const calcOptions: Array<SelectableValue<string>> = calculations.map((c) => {
    const label = c.column ? `${c.op}(${c.column})` : c.op;
    const value = JSON.stringify({ op: c.op, column: c.column ?? '' });
    return { label, value };
  });

  const add = () => {
    const first = calculations[0];
    onChange([
      ...havings,
      {
        calculateOp: first?.op ?? 'COUNT',
        column: first?.column,
        op: '>',
        value: 0,
      },
    ]);
  };

  const remove = (idx: number) => {
    onChange(havings.filter((_, i) => i !== idx));
  };

  const updateHaving = (idx: number, partial: Partial<Having>) => {
    onChange(havings.map((h, i) => (i === idx ? { ...h, ...partial } : h)));
  };

  return (
    <div>
      {havings.map((h, idx) => {
        const selectedKey = JSON.stringify({ op: h.calculateOp ?? 'COUNT', column: h.column ?? '' });
        return (
          <InlineFieldRow key={idx}>
            <InlineField label={idx === 0 ? 'Calculation' : ''} labelWidth={12}>
              <Select
                options={calcOptions}
                value={selectedKey}
                onChange={(v) => {
                  if (!v.value) {
                    return;
                  }
                  const parsed = JSON.parse(v.value) as { op: string; column: string };
                  updateHaving(idx, {
                    calculateOp: parsed.op as CalculationOp,
                    column: parsed.column ? parsed.column : undefined,
                  });
                }}
                width={20}
                placeholder="Choose calculation"
              />
            </InlineField>

            <InlineField label={idx === 0 ? 'Op' : ''} labelWidth={6}>
              <Select
                options={HAVING_OPS}
                value={h.op}
                onChange={(v) => updateHaving(idx, { op: (v.value ?? '>') as HavingOp })}
                width={8}
              />
            </InlineField>

            <InlineField label={idx === 0 ? 'Value' : ''} labelWidth={8}>
              <Input
                value={h.value === undefined ? '' : String(h.value)}
                onChange={(e) => {
                  const raw = e.currentTarget.value;
                  // Numeric if it looks like a number, else string (template variables etc.)
                  const asNum = Number(raw);
                  updateHaving(idx, {
                    value: raw === '' ? undefined : Number.isFinite(asNum) && raw.trim() !== '' && !isNaN(asNum) ? asNum : raw,
                  });
                }}
                width={16}
                placeholder="threshold"
              />
            </InlineField>

            <IconButton
              name="trash-alt"
              tooltip="Remove having"
              onClick={() => remove(idx)}
            />
          </InlineFieldRow>
        );
      })}

      <Button
        variant="secondary"
        size="sm"
        icon="plus"
        onClick={add}
        disabled={calculations.length === 0}
        tooltip={
          calculations.length === 0
            ? 'Add a calculation first'
            : 'Add a post-aggregation filter on a calculation result'
        }
      >
        Add having
      </Button>

      {/*
        Honeycomb's having spec — column is required only for ops that take a column,
        which is COLUMN_REQUIRED_OPS. The Select above already encodes this via the
        calculation's existing column.
      */}
      <input type="hidden" data-column-required-ops={Array.from(COLUMN_REQUIRED_OPS).join(',')} />
    </div>
  );
}
