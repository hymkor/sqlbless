select  'main' as schema,
        name,
        rootpage,
        sql
  from  sqlite_master
 where  type = 'table'
 union  all
select 'temp' as schema,
        name,
        rootpage,
        sql
  from  sqlite_temp_master
 where  type = 'table'
