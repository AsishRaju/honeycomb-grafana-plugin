import React from 'react';
import { SelectableValue } from '@grafana/data';
import { Field, InlineField, InlineFieldRow, Input, RadioButtonGroup } from '@grafana/ui';

import { HoneycombQuery } from '../../types';

const SLO_RESULT_TYPE_OPTIONS: Array<SelectableValue<'list' | 'single'>> = [
  { label: 'SLO List', value: 'list', description: 'List all SLOs in the dataset' },
  { label: 'Single SLO', value: 'single', description: 'Detailed compliance + budget burn for one SLO' },
];

interface Props {
  query: HoneycombQuery;
  onChange: (partial: Partial<HoneycombQuery>) => void;
}

/**
 * SLOEditor renders the SLO-specific editor controls. Active when
 * query.queryType === 'slo'.
 *
 * Design note: this editor intentionally does not enumerate available SLOs
 * because the API key may not have list permission. Users paste an SLO ID
 * (visible in the Honeycomb UI URL) for the 'single' mode.
 */
export function SLOEditor({ query, onChange }: Props) {
  const resultType = query.sloResultType ?? 'list';

  return (
    <>
      <InlineFieldRow>
        <InlineField label="Result Type" labelWidth={14}>
          <RadioButtonGroup
            options={SLO_RESULT_TYPE_OPTIONS}
            value={resultType}
            onChange={(v) => onChange({ sloResultType: v })}
          />
        </InlineField>
      </InlineFieldRow>

      {resultType === 'single' && (
        <Field
          label="SLO ID"
          description="The SLO's unique ID. Find it in the Honeycomb UI URL under /slos/<id>."
        >
          <Input
            value={query.sloId || ''}
            placeholder="e.g. abc123def"
            onChange={(e) => onChange({ sloId: e.currentTarget.value })}
            width={32}
          />
        </Field>
      )}
    </>
  );
}
