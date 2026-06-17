import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Field, FieldSet, InlineField, InlineFieldRow, Input, RadioButtonGroup } from '@grafana/ui';

import { ColumnMeta, HoneycombQuery, TracesResultType } from '../../types';
import { FiltersEditor } from './FiltersEditor';

interface Props {
  query: HoneycombQuery;
  columns: ColumnMeta[];
  onChange: (partial: Partial<HoneycombQuery>) => void;
}

const TRACE_RESULT_TYPE_OPTIONS: Array<SelectableValue<TracesResultType>> = [
  { label: 'Trace by ID', value: 'single', description: 'Fetch every span in one trace and render it as a Grafana trace view' },
  { label: 'Search', value: 'search', description: 'Find recent traces matching the filter; click a row to drill in' },
];

/**
 * TracesEditor offers two flows:
 *   - Trace by ID: paste a trace ID, render the full span tree.
 *   - Search:      filter (e.g. service.name, name, duration_ms > X) and get
 *                  a table of matching trace IDs; click any to open in
 *                  Honeycomb.
 *
 * Both modes issue an unaggregated events query (disable_series=true,
 * breakdowns of the columns we need) and rely on a backend transformer to
 * shape the response into a Grafana trace frame.
 */
export function TracesEditor({ query, columns, onChange }: Props) {
  const resultType: TracesResultType = query.tracesResultType ?? 'single';

  const columnOptions: Array<SelectableValue<string>> = columns
    .filter((c) => !c.hidden)
    .map((c) => ({ label: c.key_name, value: c.key_name, description: c.type }));

  return (
    <>
      <InlineFieldRow>
        <InlineField label="Result Type" labelWidth={20}>
          <RadioButtonGroup
            options={TRACE_RESULT_TYPE_OPTIONS}
            value={resultType}
            onChange={(v) => onChange({ tracesResultType: v })}
          />
        </InlineField>
      </InlineFieldRow>

      {resultType === 'single' && (
        <Field
          label="Trace ID"
          description="The trace ID to fetch. Supports template variables (e.g. ${trace_id})."
        >
          <Input
            value={query.traceId || ''}
            placeholder="e.g. 1234abcd5678..."
            onChange={(e) => onChange({ traceId: e.currentTarget.value })}
            width={48}
          />
        </Field>
      )}

      {resultType === 'search' && (
        <FieldSet label="Filters">
          <FiltersEditor
            filters={query.filters ?? []}
            filterCombination={query.filterCombination ?? 'AND'}
            columnOptions={columnOptions}
            onChange={(filters, filterCombination) => onChange({ filters, filterCombination })}
          />
        </FieldSet>
      )}

      <InlineFieldRow>
        <InlineField label="Limit" labelWidth={20} tooltip="Maximum number of spans (single) or traces (search) to return">
          <Input
            type="number"
            min={1}
            max={10000}
            value={query.limit ?? (resultType === 'single' ? 1000 : 50)}
            onChange={(e) => onChange({ limit: parseInt(e.currentTarget.value, 10) || 0 })}
            width={12}
          />
        </InlineField>
      </InlineFieldRow>
    </>
  );
}
