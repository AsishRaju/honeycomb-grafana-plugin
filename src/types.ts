import { DataQuery, DataSourceJsonData, SelectableValue } from '@grafana/data';

// ---------------------------------------------------------------------------
// Query model
// ---------------------------------------------------------------------------

export type QueryMode = 'timeseries' | 'table' | 'stat';
export type FilterCombination = 'AND' | 'OR';

export type CalculationOp =
  | 'COUNT'
  | 'CONCURRENCY'
  | 'SUM'
  | 'AVG'
  | 'COUNT_DISTINCT'
  | 'HEATMAP'
  | 'MAX'
  | 'MIN'
  | 'P001'
  | 'P01'
  | 'P05'
  | 'P10'
  | 'P20'
  | 'P25'
  | 'P50'
  | 'P75'
  | 'P80'
  | 'P90'
  | 'P95'
  | 'P99'
  | 'P999'
  | 'RATE_AVG'
  | 'RATE_SUM'
  | 'RATE_MAX';

export type FilterOp =
  | '='
  | '!='
  | '>'
  | '>='
  | '<'
  | '<='
  | 'starts-with'
  | 'does-not-start-with'
  | 'ends-with'
  | 'does-not-end-with'
  | 'exists'
  | 'does-not-exist'
  | 'contains'
  | 'does-not-contain'
  | 'in'
  | 'not-in';

export interface Calculation {
  op: CalculationOp;
  column?: string;
  alias?: string;
}

export interface Filter {
  column: string;
  op: FilterOp;
  value?: string | number | boolean;
}

export interface Order {
  op?: CalculationOp;
  column?: string;
  order: 'ascending' | 'descending';
}

export interface HoneycombQuery extends DataQuery {
  dataset: string;
  queryMode: QueryMode;
  calculations: Calculation[];
  filters: Filter[];
  filterCombination: FilterCombination;
  breakdowns: string[];
  orders: Order[];
  limit: number;
  granularity: number; // 0 = auto-derive
  compareTimeOffset: number; // 0 = disabled
  rawMode: boolean;
  rawJson?: string;
}

// ---------------------------------------------------------------------------
// Variable query model
// ---------------------------------------------------------------------------

export type VariableQueryType = 'datasets' | 'columns';

export interface VariableQuery {
  queryType: VariableQueryType;
  dataset?: string;
}

// ---------------------------------------------------------------------------
// Data source configuration
// ---------------------------------------------------------------------------

/**
 * Options stored in jsonData (non-secret). These are visible to the browser.
 */
export interface HoneycombDataSourceOptions extends DataSourceJsonData {
  /**
   * Honeycomb API base URL.
   * Defaults to https://api.honeycomb.io
   * Use https://api.eu1.honeycomb.io for EU accounts.
   */
  apiUrl?: string;

  /** L1 cache TTL in minutes (query shape → query_id). Default: 30. */
  cacheTtlL1Minutes?: number;

  /** L2 cache TTL in minutes (execution key → query_result_id). Default: 10. */
  cacheTtlL2Minutes?: number;

  /** L3 cache TTL in minutes (completed query results). Default: 120. */
  cacheTtlL3Minutes?: number;
}

/**
 * Options stored in secureJsonData (encrypted). NEVER sent to the browser.
 */
export interface HoneycombSecureJsonData {
  apiKey?: string;
}

// ---------------------------------------------------------------------------
// Resource API responses (from backend resource handlers)
// ---------------------------------------------------------------------------

export const ALL_DATASETS_SLUG = '__all__';

export interface DatasetMeta {
  name: string;
  slug: string;
  description?: string;
  created_at?: string;
}

export interface ColumnMeta {
  id: string;
  key_name: string;
  type: 'string' | 'float' | 'integer' | 'boolean';
  hidden: boolean;
  description?: string;
}

// ---------------------------------------------------------------------------
// UI helpers
// ---------------------------------------------------------------------------

export const CALCULATION_OPS: Array<SelectableValue<CalculationOp>> = [
  { label: 'COUNT', value: 'COUNT', description: 'Number of matching events' },
  { label: 'CONCURRENCY', value: 'CONCURRENCY' },
  { label: 'SUM', value: 'SUM', description: 'Sum of a numeric column' },
  { label: 'AVG', value: 'AVG', description: 'Average of a numeric column' },
  { label: 'COUNT_DISTINCT', value: 'COUNT_DISTINCT', description: 'Distinct values of a column' },
  { label: 'MAX', value: 'MAX' },
  { label: 'MIN', value: 'MIN' },
  { label: 'P50', value: 'P50', description: 'Median' },
  { label: 'P75', value: 'P75' },
  { label: 'P90', value: 'P90' },
  { label: 'P95', value: 'P95' },
  { label: 'P99', value: 'P99' },
  { label: 'P99.9', value: 'P999' },
  { label: 'HEATMAP', value: 'HEATMAP' },
  { label: 'RATE_AVG', value: 'RATE_AVG' },
  { label: 'RATE_SUM', value: 'RATE_SUM' },
  { label: 'RATE_MAX', value: 'RATE_MAX' },
  { label: 'P0.01', value: 'P001' },
  { label: 'P0.1', value: 'P01' },
  { label: 'P5', value: 'P05' },
  { label: 'P10', value: 'P10' },
  { label: 'P20', value: 'P20' },
  { label: 'P25', value: 'P25' },
  { label: 'P80', value: 'P80' },
];

/** Operations that require a column parameter. */
export const COLUMN_REQUIRED_OPS = new Set<CalculationOp>([
  'SUM', 'AVG', 'COUNT_DISTINCT', 'HEATMAP', 'MAX', 'MIN',
  'P001', 'P01', 'P05', 'P10', 'P20', 'P25', 'P50', 'P75',
  'P80', 'P90', 'P95', 'P99', 'P999', 'RATE_AVG', 'RATE_SUM', 'RATE_MAX',
]);

export const FILTER_OPS: Array<SelectableValue<FilterOp>> = [
  { label: '=', value: '=' },
  { label: '≠', value: '!=' },
  { label: '>', value: '>' },
  { label: '≥', value: '>=' },
  { label: '<', value: '<' },
  { label: '≤', value: '<=' },
  { label: 'starts-with', value: 'starts-with' },
  { label: 'does-not-start-with', value: 'does-not-start-with' },
  { label: 'ends-with', value: 'ends-with' },
  { label: 'does-not-end-with', value: 'does-not-end-with' },
  { label: 'exists', value: 'exists' },
  { label: 'does-not-exist', value: 'does-not-exist' },
  { label: 'contains', value: 'contains' },
  { label: 'does-not-contain', value: 'does-not-contain' },
  { label: 'in', value: 'in' },
  { label: 'not-in', value: 'not-in' },
];

/** Operations that do not take a value argument. */
export const NO_VALUE_FILTER_OPS = new Set<FilterOp>(['exists', 'does-not-exist']);

export const QUERY_MODE_OPTIONS: Array<SelectableValue<QueryMode>> = [
  { label: 'Time series', value: 'timeseries', description: 'One series per breakdown group' },
  { label: 'Table', value: 'table', description: 'Summary results as a table' },
  { label: 'Stat', value: 'stat', description: 'Single aggregated value' },
];

export const DEFAULT_API_URL = 'https://api.honeycomb.io';
export const EU_API_URL = 'https://api.eu1.honeycomb.io';
