import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceInstanceSettings,
  MetricFindValue,
  ScopedVars,
  dateTime,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import {
  ColumnMeta,
  DatasetMeta,
  HoneycombDataSourceOptions,
  HoneycombQuery,
  VariableQuery,
} from './types';
import { defaultQuery } from './defaults';

/**
 * HoneycombDataSource is the frontend counterpart to the Go backend datasource.
 *
 * Its responsibilities are limited to:
 * 1. Applying template variable substitution before queries are sent to the backend.
 * 2. Filtering out hidden and empty queries before they reach the backend.
 * 3. Providing metadata (datasets, columns) via backend resource handlers.
 * 4. Supporting dashboard variable queries.
 *
 * ALL Honeycomb API calls (including secrets) go through the backend.
 */
export class HoneycombDataSource extends DataSourceWithBackend<HoneycombQuery, HoneycombDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<HoneycombDataSourceOptions>) {
    super(instanceSettings);
    this.annotations = {};
  }

  // ---------------------------------------------------------------------------
  // Query filtering and variable substitution
  // ---------------------------------------------------------------------------

  /**
   * filterQuery is called by Grafana before running each query. Return false
   * to skip the query entirely (avoids sending empty/invalid queries to Honeycomb).
   */
  filterQuery(query: HoneycombQuery): boolean {
    if (!query.dataset?.trim()) {
      return false;
    }
    if (!query.rawMode && (!query.calculations || query.calculations.length === 0)) {
      return false;
    }
    if (query.rawMode && !query.rawJson?.trim()) {
      return false;
    }
    return true;
  }

  /**
   * applyTemplateVariables substitutes Grafana dashboard variables in all
   * string fields of the query before it is sent to the backend.
   */
  applyTemplateVariables(query: HoneycombQuery, scopedVars: ScopedVars): HoneycombQuery {
    const replace = (s: string) => getTemplateSrv().replace(s, scopedVars);

    return {
      ...query,
      dataset: replace(query.dataset ?? ''),
      breakdowns: (query.breakdowns ?? []).map(replace),
      filters: (query.filters ?? []).map((f) => ({
        ...f,
        column: replace(f.column),
        value: typeof f.value === 'string' ? replace(f.value) : f.value,
      })),
      rawJson: query.rawJson ? replace(query.rawJson) : query.rawJson,
    };
  }

  /**
   * getDefaultQuery returns the default query shown when a new panel is added.
   */
  getDefaultQuery(): Partial<HoneycombQuery> {
    return defaultQuery();
  }

  // ---------------------------------------------------------------------------
  // Dashboard variable support
  // ---------------------------------------------------------------------------

  /**
   * metricFindQuery handles template variable queries.
   * Returns selectable values based on the variable query type:
   *   - queryType: 'datasets' → list of dataset slugs
   *   - queryType: 'columns'  → list of column names for the given dataset
   */
  async metricFindQuery(query: VariableQuery | string): Promise<MetricFindValue[]> {
    const vq = typeof query === 'string' ? parseVariableQueryString(query) : query;

    switch (vq.queryType) {
      case 'datasets': {
        const datasets = await this.listDatasets();
        return datasets.map((d) => ({ text: d.name, value: d.slug }));
      }
      case 'columns': {
        if (!vq.dataset) {
          return [];
        }
        const cols = await this.listColumns(vq.dataset);
        return cols.filter((c) => !c.hidden).map((c) => ({ text: c.key_name, value: c.key_name }));
      }
      default:
        return [];
    }
  }

  // ---------------------------------------------------------------------------
  // Metadata resource calls (backed by the Go resource handler)
  // ---------------------------------------------------------------------------

  /**
   * listDatasets fetches all available Honeycomb datasets.
   * Results are cached for 5 minutes on the backend.
   */
  async listDatasets(): Promise<DatasetMeta[]> {
    const response = await this.getResource('datasets');
    return response as DatasetMeta[];
  }

  /**
   * listColumns fetches column metadata for the given dataset.
   * Results are cached for 5 minutes on the backend.
   */
  async listColumns(dataset: string): Promise<ColumnMeta[]> {
    const response = await this.getResource(`columns?dataset=${encodeURIComponent(dataset)}`);
    return response as ColumnMeta[];
  }

  // ---------------------------------------------------------------------------
  // Annotation support
  // ---------------------------------------------------------------------------

  /**
   * Annotations are handled via the standard QueryData path. Grafana routes
   * annotation queries to QueryData with a special annotationQuery flag.
   * The backend's transform.ToAnnotationFrames produces the required frame format.
   */
  annotations = {};
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * parseVariableQueryString parses a simple legacy string variable query of
 * the form "datasets" or "columns:<dataset-slug>".
 */
function parseVariableQueryString(query: string): VariableQuery {
  const trimmed = query.trim().toLowerCase();
  if (trimmed === 'datasets') {
    return { queryType: 'datasets' };
  }
  if (trimmed.startsWith('columns:')) {
    return { queryType: 'columns', dataset: query.slice('columns:'.length).trim() };
  }
  return { queryType: 'datasets' };
}
