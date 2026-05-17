import React, { useState, useEffect } from 'react';
import { SelectableValue } from '@grafana/data';
import { Field, InlineField, InlineFieldRow, Select } from '@grafana/ui';

import { HoneycombDataSource } from '../../datasource';
import { ALL_DATASETS_SLUG, DatasetMeta, VariableQuery, VariableQueryType } from '../../types';

interface Props {
  query: VariableQuery | string;
  onChange: (query: VariableQuery, definition: string) => void;
  datasource: HoneycombDataSource;
}

const QUERY_TYPE_OPTIONS: Array<SelectableValue<VariableQueryType>> = [
  { label: 'Datasets', value: 'datasets', description: 'List available Honeycomb datasets' },
  { label: 'Columns', value: 'columns', description: 'List columns for a dataset' },
];

/**
 * VariableQueryEditor is shown when a user creates a Grafana dashboard variable
 * backed by this datasource.
 *
 * Supported variable query types:
 * - datasets: returns slug/name pairs for all accessible datasets
 * - columns:  returns column names for a given dataset
 *
 * The selected values can be used as $dataset or $column template variables
 * in query editors.
 */
export function VariableQueryEditor({ query, onChange, datasource }: Props) {
  // Normalise legacy string format.
  const parsed: VariableQuery = typeof query === 'string' ? parseString(query) : query;

  const allDatasetsOption: SelectableValue<string> = { label: 'All Datasets', value: ALL_DATASETS_SLUG };

  const [datasets, setDatasets] = useState<Array<SelectableValue<string>>>([allDatasetsOption]);

  useEffect(() => {
    datasource
      .listDatasets()
      .then((ds: DatasetMeta[]) => {
        setDatasets([allDatasetsOption, ...ds.map((d) => ({ label: d.name, value: d.slug }))]);
      })
      .catch(() => setDatasets([allDatasetsOption]));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const update = (partial: Partial<VariableQuery>) => {
    const next: VariableQuery = { ...parsed, ...partial };
    onChange(next, buildDefinition(next));
  };

  return (
    <div>
      <InlineFieldRow>
        <InlineField label="Query type" labelWidth={12}>
          <Select
            options={QUERY_TYPE_OPTIONS}
            value={parsed.queryType}
            onChange={(v) => update({ queryType: v.value ?? 'datasets' })}
            width={20}
          />
        </InlineField>
      </InlineFieldRow>

      {parsed.queryType === 'columns' && (
        <InlineFieldRow>
          <InlineField label="Dataset" labelWidth={12}>
            <Select
              options={datasets}
              value={parsed.dataset}
              onChange={(v) => update({ dataset: v.value })}
              allowCustomValue
              placeholder="Select or type dataset slug"
              width={24}
            />
          </InlineField>
        </InlineFieldRow>
      )}

      <div style={{ color: 'var(--text-muted)', fontSize: '12px', marginTop: '8px' }}>
        {parsed.queryType === 'datasets' && 'Returns a list of dataset slugs for use in panel queries.'}
        {parsed.queryType === 'columns' &&
          parsed.dataset &&
          `Returns column names for dataset "${parsed.dataset}". Use $variable as a column name in queries.`}
      </div>
    </div>
  );
}

function parseString(query: string): VariableQuery {
  const trimmed = query.trim().toLowerCase();
  if (trimmed.startsWith('columns:')) {
    return { queryType: 'columns', dataset: query.slice('columns:'.length).trim() };
  }
  return { queryType: 'datasets' };
}

function buildDefinition(query: VariableQuery): string {
  if (query.queryType === 'columns' && query.dataset) {
    return `columns:${query.dataset}`;
  }
  return 'datasets';
}
