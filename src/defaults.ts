import { HoneycombQuery } from './types';

/**
 * defaultQuery returns the initial query shown when a user adds a new panel.
 * Deliberately minimal: one COUNT calculation, sensible defaults, no filters.
 */
export function defaultQuery(): Partial<HoneycombQuery> {
  return {
    dataset: '',
    queryMode: 'timeseries',
    calculations: [{ op: 'COUNT' }],
    filters: [],
    filterCombination: 'AND',
    breakdowns: [],
    orders: [],
    limit: 100,
    granularity: 0,
    compareTimeOffset: 0,
    rawMode: false,
  };
}
