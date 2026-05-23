select  column_id,
        column_name,
        case nullable
            when 'Y' then 'NULL'
            else 'NOT NULL'
        end as "NULL?",
        case
            when data_type = 'NUMBER' then data_type
            when data_type = 'DATE' then data_type
            when data_type like 'TIMESTAMP%' then data_type
            else data_type || '(' || data_length || ')'
        end as "TYPE"
  from  all_tab_columns c
 where  c.table_name = case
            when :1 like '%.%' then upper(regexp_substr(:1,'[^.]+$'))
            else UPPER(:1)
        end
   and  c.owner = case
            when :1 like '%.%' then upper(regexp_substr(:1,'^[^.]+'))
            else sys_context('USERENV','CURRENT_SCHEMA')
        end
 order  by column_id
