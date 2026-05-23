select  n.nspname || '.' || c.relname as full_name,
        n.nspname as table_schema,
        c.relname as table_name,
        case c.relkind
            when 'r' then 'BASE TABLE'
            when 'p' then 'PARTITIONED TABLE'
            when 'v' then 'VIEW'
            when 'm' then 'MATERIALIZED VIEW'
            when 'f' then 'FOREIGN TABLE'
        end as table_type,
        c.reltuples::bigint as estimated_rows,
        obj_description(c.oid,'pg_class') as remarks
  from  pg_class c
  join  pg_namespace n
    on  n.oid = c.relnamespace
 where  c.relkind in ('r','p','v','m','f')
 order  by
        case n.nspname
            when 'pg_catalog' then 9
            when 'information_schema' then 8
            else 0
        end,
        case c.relkind
            when 'r' then 0
            when 'p' then 1
            else 9
        end,
        n.nspname,
        c.relname
