import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Button, IconButton, InlineField, InlineFieldRow, Select } from '@grafana/ui';

interface Props {
  breakdowns: string[];
  columnOptions: Array<SelectableValue<string>>;
  onChange: (breakdowns: string[]) => void;
}

/**
 * GroupByEditor is the high-cardinality-first breakdown configuration.
 *
 * "Group by" in Grafana maps directly to Honeycomb's "breakdowns" — the set of
 * columns by which events are grouped. Each unique combination of breakdown
 * values produces a separate series in timeseries mode or a separate row in
 * table mode.
 *
 * Honeycomb supports up to 100 breakdowns. More than ~20 will generate many
 * series; use Limit + Order by to control cardinality.
 */
export function GroupByEditor({ breakdowns, columnOptions, onChange }: Props) {
  const add = () => {
    onChange([...breakdowns, '']);
  };

  const remove = (idx: number) => {
    onChange(breakdowns.filter((_, i) => i !== idx));
  };

  const update = (idx: number, value: string) => {
    onChange(breakdowns.map((b, i) => (i === idx ? value : b)));
  };

  return (
    <div>
      {breakdowns.map((breakdown, idx) => (
        <InlineFieldRow key={idx}>
          <InlineField label={idx === 0 ? 'Column' : ''} labelWidth={8}>
            <Select
              options={columnOptions}
              value={breakdown || null}
              onChange={(v) => update(idx, v.value ?? '')}
              allowCustomValue
              placeholder="column name"
              width={24}
            />
          </InlineField>
          <IconButton name="trash-alt" tooltip="Remove group by column" onClick={() => remove(idx)} />
        </InlineFieldRow>
      ))}

      {breakdowns.length === 0 && (
        <div style={{ color: 'var(--text-muted)', fontSize: '12px', marginBottom: '8px' }}>
          No breakdowns — results are aggregated across all events.
        </div>
      )}

      <Button
        variant="secondary"
        size="sm"
        icon="plus"
        onClick={add}
        disabled={breakdowns.length >= 100}
      >
        Add group by column
      </Button>
    </div>
  );
}
