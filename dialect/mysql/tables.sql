select  concat(table_schema,'.',table_name) as FULL_NAME,
        tables.*
  from  information_schema.tables
 where  table_type = 'BASE TABLE'
   and  table_schema not in ('mysql', 'information_schema', 'performance_schema', 'sys')
