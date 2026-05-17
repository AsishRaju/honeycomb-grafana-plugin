import React, { useCallback, useEffect, useState } from 'react';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import {
  Button,
  Field,
  FieldSet,
  InlineField,
  InlineFieldRow,
  Input,
  Select,
  Spinner,
  TextArea,
  ToolbarButton,
  useTheme2,
} from '@grafana/ui';
import { css } from '@emotion/css';

import { HoneycombDataSource } from '../../datasource';
import {
  ALL_DATASETS_SLUG,
  CalculationOp,
  ColumnMeta,
  HoneycombDataSourceOptions,
  HoneycombQuery,
  QUERY_MODE_OPTIONS,
  QueryMode,
} from '../../types';
import { defaultQuery } from '../../defaults';
import { CalculationsEditor } from './CalculationsEditor';
import { FiltersEditor } from './FiltersEditor';
import { GroupByEditor } from './GroupByEditor';
import { OrderByEditor } from './OrderByEditor';

type Props = QueryEditorProps<HoneycombDataSource, HoneycombQuery, HoneycombDataSourceOptions>;

/**
 * QueryEditor is the main query builder UI for the Honeycomb datasource.
 *
 * It provides two modes:
 * 1. Visual builder: dataset picker, calculations, filters, group-by, order-by, limit, granularity.
 * 2. Raw JSON mode: a textarea for pasting raw Honeycomb Query API JSON.
 *
 * Template variables (${var_name}) are supported in any string input.
 */
export function QueryEditor({ datasource, query, onChange, onRunQuery }: Props) {
  const theme = useTheme2();
  const styles = getStyles(theme);

  const q: HoneycombQuery = { ...defaultQuery(), ...query } as HoneycombQuery;

  const allDatasetsOption: SelectableValue<string> = { label: 'All Datasets', value: ALL_DATASETS_SLUG, description: 'Query across all datasets in the environment' };

  const [datasets, setDatasets] = useState<Array<SelectableValue<string>>>([allDatasetsOption]);
  const [columns, setColumns] = useState<ColumnMeta[]>([]);
  const [loadingDatasets, setLoadingDatasets] = useState(false);
  const [loadingColumns, setLoadingColumns] = useState(false);
  const [honeycombUrl, setHoneycombUrl] = useState<string | undefined>();

  // Load datasets on mount.
  useEffect(() => {
    setLoadingDatasets(true);
    datasource
      .listDatasets()
      .then((ds) => {
        setDatasets([allDatasetsOption, ...ds.map((d) => ({ label: d.name, value: d.slug, description: d.description }))]);
      })
      .catch(() => setDatasets([allDatasetsOption]))
      .finally(() => setLoadingDatasets(false));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Load columns when dataset changes.
  useEffect(() => {
    if (!q.dataset) {
      setColumns([]);
      return;
    }
    setLoadingColumns(true);
    datasource
      .listColumns(q.dataset)
      .then(setColumns)
      .catch(() => setColumns([]))
      .finally(() => setLoadingColumns(false));
  }, [q.dataset]); // eslint-disable-line react-hooks/exhaustive-deps

  const update = useCallback(
    (partial: Partial<HoneycombQuery>) => {
      onChange({ ...q, ...partial });
    },
    [q, onChange]
  );

  const handleRunQuery = () => onRunQuery();

  // Build column options for selects from loaded column metadata.
  const columnOptions: Array<SelectableValue<string>> = columns
    .filter((c) => !c.hidden)
    .map((c) => ({
      label: c.key_name,
      value: c.key_name,
      description: c.type,
    }));

  if (q.rawMode) {
    return (
      <div className={styles.wrapper}>
        <InlineFieldRow>
          <InlineField label="Raw JSON mode" labelWidth={16}>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => update({ rawMode: false })}
            >
              Switch to visual editor
            </Button>
          </InlineField>
        </InlineFieldRow>

        <Field
          label="Honeycomb Query JSON"
          description="Paste a raw Honeycomb Query API JSON object. Template variables are supported."
        >
          <TextArea
            rows={12}
            value={q.rawJson || ''}
            onChange={(e) => update({ rawJson: e.currentTarget.value })}
            onBlur={handleRunQuery}
            placeholder='{"calculations":[{"op":"COUNT"}],"breakdowns":["service.name"]}'
          />
        </Field>
      </div>
    );
  }

  return (
    <div className={styles.wrapper}>
      {/* Dataset + mode row */}
      <InlineFieldRow>
        <InlineField label="Dataset" labelWidth={12} grow>
          {loadingDatasets ? (
            <Spinner />
          ) : (
            <Select
              options={datasets}
              value={q.dataset}
              onChange={(v) => {
                update({ dataset: v.value ?? '' });
              }}
              allowCustomValue
              placeholder="Select dataset or type slug"
              width={24}
            />
          )}
        </InlineField>

        <InlineField label="Query mode" labelWidth={12}>
          <Select
            options={QUERY_MODE_OPTIONS}
            value={q.queryMode}
            onChange={(v) => update({ queryMode: (v.value as QueryMode) ?? 'timeseries' })}
            width={16}
          />
        </InlineField>

        <InlineField>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => update({ rawMode: true })}
            title="Switch to raw JSON mode"
          >
            {'{ } Raw'}
          </Button>
        </InlineField>
      </InlineFieldRow>

      {/* Calculations */}
      <FieldSet label="Calculations">
        <CalculationsEditor
          calculations={q.calculations ?? []}
          columnOptions={columnOptions}
          loadingColumns={loadingColumns}
          onChange={(calculations) => update({ calculations })}
        />
      </FieldSet>

      {/* Filters */}
      <FieldSet label="Filters">
        <FiltersEditor
          filters={q.filters ?? []}
          filterCombination={q.filterCombination ?? 'AND'}
          columnOptions={columnOptions}
          onChange={(filters, filterCombination) => update({ filters, filterCombination })}
        />
      </FieldSet>

      {/* Group by */}
      <FieldSet label="Group by (Breakdowns)">
        <GroupByEditor
          breakdowns={q.breakdowns ?? []}
          columnOptions={columnOptions}
          onChange={(breakdowns) => update({ breakdowns })}
        />
      </FieldSet>

      {/* Order by */}
      <FieldSet label="Order by">
        <OrderByEditor
          orders={q.orders ?? []}
          calculations={q.calculations ?? []}
          breakdowns={q.breakdowns ?? []}
          onChange={(orders) => update({ orders })}
        />
      </FieldSet>

      {/* Options row: limit, granularity */}
      <InlineFieldRow>
        <InlineField label="Limit" labelWidth={10} tooltip="Maximum number of result groups (1–10000)">
          <Input
            type="number"
            min={1}
            max={10000}
            value={q.limit ?? 100}
            onChange={(e) => update({ limit: parseInt(e.currentTarget.value, 10) || 100 })}
            onBlur={handleRunQuery}
            width={10}
          />
        </InlineField>

        <InlineField
          label="Granularity (s)"
          labelWidth={14}
          tooltip="Time resolution in seconds. 0 = auto-derive from panel time range."
        >
          <Input
            type="number"
            min={0}
            value={q.granularity ?? 0}
            onChange={(e) => update({ granularity: parseInt(e.currentTarget.value, 10) || 0 })}
            onBlur={handleRunQuery}
            placeholder="0 (auto)"
            width={10}
          />
        </InlineField>
      </InlineFieldRow>

      {/* Run button */}
      <div className={styles.runRow}>
        <ToolbarButton
          variant="primary"
          onClick={handleRunQuery}
          icon="play"
        >
          Run query
        </ToolbarButton>
      </div>
    </div>
  );
}

function getStyles(theme: ReturnType<typeof useTheme2>) {
  return {
    wrapper: css`
      display: flex;
      flex-direction: column;
      gap: ${theme.spacing(1)};
    `,
    runRow: css`
      display: flex;
      justify-content: flex-end;
      margin-top: ${theme.spacing(1)};
    `,
  };
}
