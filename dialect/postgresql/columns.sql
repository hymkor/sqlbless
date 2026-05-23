select  attname as "NAME",
        case 
            when attnotnull then 'NOT NULL'
            else 'NULL'
        end as "NULL?",
        format_type(atttypid, atttypmod) as "TYPE"
  from  pg_attribute
 where  attrelid = to_regclass($1)::oid
   and  attnum > 0
   and  not attisdropped
 order  by attnum
