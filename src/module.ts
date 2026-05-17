import { DataSourcePlugin } from '@grafana/data';

import { HoneycombDataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor/ConfigEditor';
import { QueryEditor } from './components/QueryEditor/QueryEditor';
import { VariableQueryEditor } from './components/VariableQueryEditor/VariableQueryEditor';
import { HoneycombQuery, HoneycombDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<HoneycombDataSource, HoneycombQuery, HoneycombDataSourceOptions>(
  HoneycombDataSource
)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor)
  .setVariableQueryEditor(VariableQueryEditor);
