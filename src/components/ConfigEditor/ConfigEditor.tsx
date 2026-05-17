import React, { ChangeEvent } from 'react';
import { DataSourcePluginOptionsEditorProps, SelectableValue } from '@grafana/data';
import { Field, Input, SecretInput, Select, FieldSet, Alert } from '@grafana/ui';

import { HoneycombDataSourceOptions, HoneycombSecureJsonData, DEFAULT_API_URL, EU_API_URL } from '../../types';

type Props = DataSourcePluginOptionsEditorProps<HoneycombDataSourceOptions, HoneycombSecureJsonData>;

const API_URL_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'US (api.honeycomb.io)', value: DEFAULT_API_URL },
  { label: 'EU (api.eu1.honeycomb.io)', value: EU_API_URL },
  { label: 'Custom', value: 'custom' },
];

/**
 * ConfigEditor is shown on the data source configuration page in Grafana.
 *
 * The API key is stored in secureJsonData and never returned to the browser
 * after being saved. The Grafana backend encrypts it at rest.
 */
export function ConfigEditor({ options, onOptionsChange }: Props) {
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const apiUrl = jsonData.apiUrl || DEFAULT_API_URL;
  const isCustomUrl = apiUrl !== DEFAULT_API_URL && apiUrl !== EU_API_URL;
  const selectedPreset = isCustomUrl ? 'custom' : apiUrl;

  const onApiUrlPresetChange = (selected: SelectableValue<string>) => {
    if (selected.value !== 'custom') {
      onOptionsChange({
        ...options,
        jsonData: { ...jsonData, apiUrl: selected.value },
      });
    }
  };

  const onCustomUrlChange = (e: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: { ...jsonData, apiUrl: e.target.value },
    });
  };

  const onApiKeyChange = (e: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: { ...secureJsonData, apiKey: e.target.value },
    });
  };

  const onApiKeyReset = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: { ...secureJsonFields, apiKey: false },
      secureJsonData: { ...secureJsonData, apiKey: '' },
    });
  };

  return (
    <div>
      <FieldSet label="Connection">
        <Field
          label="API Region"
          description="Select your Honeycomb account region, or enter a custom API URL."
        >
          <Select
            options={API_URL_OPTIONS}
            value={selectedPreset}
            onChange={onApiUrlPresetChange}
            width={32}
          />
        </Field>

        {isCustomUrl && (
          <Field label="Custom API URL">
            <Input
              placeholder="https://api.honeycomb.io"
              value={apiUrl}
              onChange={onCustomUrlChange}
              width={40}
            />
          </Field>
        )}

        <Field
          label="API Key"
          description={
            <>
              Your Honeycomb Configuration API Key with{' '}
              <strong>Manage Queries and Columns</strong> and <strong>Run Queries</strong> permissions.{' '}
              The key is stored encrypted and is never returned to the browser.
            </>
          }
        >
          <SecretInput
            isConfigured={Boolean(secureJsonFields?.apiKey)}
            value={secureJsonData?.apiKey || ''}
            placeholder="your-honeycomb-api-key"
            width={40}
            onReset={onApiKeyReset}
            onChange={onApiKeyChange}
          />
        </Field>
      </FieldSet>

      <Alert title="API Rate Limits" severity="info">
        Honeycomb limits query execution to <strong>10 requests per minute</strong>. This plugin
        uses aggressive caching (up to 24 hours for completed results) and a token-bucket rate
        limiter to stay within this limit. Dashboards with many panels sharing the same query will
        automatically coalesce requests. See the{' '}
        <a
          href="https://github.com/honeycombio/grafana-honeycomb-datasource/blob/main/docs/adr/ADR-002-caching-strategy.md"
          target="_blank"
          rel="noopener noreferrer"
        >
          caching documentation
        </a>{' '}
        for details.
      </Alert>
    </div>
  );
}
