select  c.column_id as "ID",
        c.name as "NAME",
        case c.is_nullable
            when 1 then 'NULL'
            else 'NOT NULL'
        end as "NULL?",
        case
            when t.name in ('char','varchar','nchar','nvarchar',
                            'binary','varbinary')
             and c.max_length > 0
            then t.name + '(' +
                 convert(varchar,
                    case
                        when t.name in ('nchar','nvarchar')
                        then c.max_length/2
                        else c.max_length
                    end
                 ) + ')'
             else t.name
        end as "TYPE"
  from  sys.columns c,
        sys.objects o,
        sys.types   t,
        sys.schemas s
 where  c.object_id = o.object_id
   and  o.name = case
            when @p1 like '%.%' then parsename(@p1,1)
            else @p1
        end
   and  o.schema_id = s.schema_id
   and  s.name = case
            when @p1 like '%.%' then parsename(@p1,2)
            else schema_name()
        end
   and  c.user_type_id = t.user_type_id
 order  by c.column_id
