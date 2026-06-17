import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Field, FieldSet, InlineField, InlineFieldRow, Input, MultiSelect } from '@grafana/ui';

import { ColumnMeta, HoneycombQuery } from '../../types';
import { FiltersEditor } from './FiltersEditor';

interface Props {
  query: HoneycombQuery;
  columns: ColumnMeta[];
  onChange: (partial: Partial<HoneycombQuery>) => void;
  onRunQuery: () => void;
}

/**
 * LogsEditor renders the dedicated logs UI: a filter panel + an attributes
 * picker + a limit. No calculations or breakdowns — the backend issues an
 * unaggregated events query, then the transformer renders rows as log lines.
 *
 * Compatible with Grafana's Logs panel and the Explore logs view because
 * plugin.json declares logs: true and the backend emits FrameTypeLogLines.
 */
export function LogsEditor({ query, columns, onChange, onRunQuery }: Props) {
  const columnOptions: Array<SelectableValue<string>> = columns
    .filter((c) => !c.hidden)
    .map((c) => ({ label: c.key_name, value: c.key_name, description: c.type }));

  const handleRunQuery = () => onRunQuery();

  return (
    <>
      <FieldSet label="Filters">
        <FiltersEditor
          filters={query.filters ?? []}
          filterCombination={query.filterCombination ?? 'AND'}
          columnOptions={columnOptions}
          onChange={(filters, filterCombination) => onChange({ filters, filterCombination })}
        />
      </FieldSet>

      <Field
        label="Show attributes"
        description="Which columns to include as inline attributes in each log line. Leave empty to include all non-hidden columns."
      >
        <MultiSelect
          options={columnOptions}
          value={query.logsAttributes ?? []}
          onChange={(values) => onChange({ logsAttributes: values.map((v) => v.value as string).filter(Boolean) })}
          allowCustomValue
          placeholder="Choose columns or leave empty for all"
          width={64}
        />
      </Field>

      <InlineFieldRow>
        <InlineField label="Limit" labelWidth={20} tooltip="Maximum number of log rows to return (1–10000)">
          <Input
            type="number"
            min={1}
            max={10000}
            value={query.limit ?? 1000}
            onChange={(e) => onChange({ limit: parseInt(e.currentTarget.value, 10) || 1000 })}
            onBlur={handleRunQuery}
            width={12}
          />
        </InlineField>
      </InlineFieldRow>
    </>
  );
}
