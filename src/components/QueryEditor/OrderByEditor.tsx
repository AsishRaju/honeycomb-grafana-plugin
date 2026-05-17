import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Button, IconButton, InlineField, InlineFieldRow, RadioButtonGroup, Select } from '@grafana/ui';

import { Calculation, CalculationOp, CALCULATION_OPS, Order } from '../../types';

interface Props {
  orders: Order[];
  calculations: Calculation[];
  breakdowns: string[];
  onChange: (orders: Order[]) => void;
}

const ORDER_DIRECTION_OPTIONS = [
  { label: 'Desc', value: 'descending' as const },
  { label: 'Asc', value: 'ascending' as const },
];

/**
 * OrderByEditor lets users configure sort order for Honeycomb results.
 *
 * Honeycomb requires that order terms reference either:
 * - A breakdown column (sort by group value), or
 * - A calculation operation (sort by metric value).
 *
 * This editor builds combined options from both breakdown columns and
 * calculation ops so the user doesn't need to distinguish them manually.
 */
export function OrderByEditor({ orders, calculations, breakdowns, onChange }: Props) {
  // Build combined options: breakdown columns first, then calculations.
  const calcOptions: Array<SelectableValue<string>> = calculations.map((c) => {
    const label = c.alias || (c.column ? `${c.op}(${c.column})` : c.op);
    return { label: `[calc] ${label}`, value: c.op, description: `${c.op} calculation` };
  });

  const breakdownOptions: Array<SelectableValue<string>> = breakdowns.map((b) => ({
    label: `[group] ${b}`,
    value: b,
    description: 'breakdown column',
  }));

  const allOptions = [...breakdownOptions, ...calcOptions];

  const add = () => {
    const firstOption = allOptions[0];
    if (!firstOption) {
      return;
    }
    // Determine if the first option is a calculation or breakdown.
    const isCalc = calculations.some((c) => c.op === firstOption.value);
    onChange([
      ...orders,
      isCalc
        ? { op: firstOption.value as CalculationOp, order: 'descending' }
        : { column: firstOption.value, order: 'descending' },
    ]);
  };

  const remove = (idx: number) => {
    onChange(orders.filter((_, i) => i !== idx));
  };

  const update = (idx: number, partial: Partial<Order>) => {
    onChange(orders.map((o, i) => (i === idx ? { ...o, ...partial } : o)));
  };

  const getValue = (order: Order): string => order.column ?? order.op ?? '';

  const handleTermChange = (idx: number, value: string) => {
    const isCalc = calculations.some((c) => c.op === value);
    if (isCalc) {
      update(idx, { op: value as CalculationOp, column: undefined });
    } else {
      update(idx, { column: value, op: undefined });
    }
  };

  return (
    <div>
      {orders.map((order, idx) => (
        <InlineFieldRow key={idx}>
          <InlineField label={idx === 0 ? 'Sort by' : ''} labelWidth={8}>
            <Select
              options={allOptions}
              value={getValue(order)}
              onChange={(v) => handleTermChange(idx, v.value ?? '')}
              width={24}
            />
          </InlineField>

          <InlineField label={idx === 0 ? 'Direction' : ''} labelWidth={10}>
            <RadioButtonGroup
              options={ORDER_DIRECTION_OPTIONS}
              value={order.order}
              onChange={(v) => update(idx, { order: v as 'ascending' | 'descending' })}
            />
          </InlineField>

          <IconButton name="trash-alt" tooltip="Remove order" onClick={() => remove(idx)} />
        </InlineFieldRow>
      ))}

      {orders.length === 0 && (
        <div style={{ color: 'var(--text-muted)', fontSize: '12px', marginBottom: '8px' }}>
          No sort order — results are returned in Honeycomb&apos;s default order.
        </div>
      )}

      <Button
        variant="secondary"
        size="sm"
        icon="plus"
        onClick={add}
        disabled={orders.length >= 10 || allOptions.length === 0}
      >
        Add sort
      </Button>
    </div>
  );
}
