select  ordinal_position as "ID",
        column_name as "NAME",
        case
            when character_maximum_length is not null then 
                 concat(data_type,'(',character_maximum_length,')')
            when datetime_precision is not null then
                 concat(data_type,'(',datetime_precision,')')
        else
            data_type
        end as "TYPE",
        case is_nullable
            when "YES" then 'NULL'
            else 'NOT NULL'
        end as "NULL?"
  from  information_schema.columns,
        (select ? as v1) arg
 where  table_name   = REGEXP_REPLACE(v1,'^[^\\.]*\\.','')
   and  table_schema = case
            when v1 like '%.%' then regexp_replace(v1,'\\.[^\\.]*$','')
            else database()
        end
 order  by ordinal_position
