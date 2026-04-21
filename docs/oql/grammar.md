# SovereignQL (OQL) Grammar Specification

OQL is a pipe-based query language designed for tactical security analytics. It follows a structure similar to Splunk's SPL or Kusto's KQL, optimized for high-performance event processing.

## Query Structure

A query consists of an optional initial search expression followed by a series of piped commands.

```
query = [ search_expr ] { "|" command }
```

## Search Expressions

The initial search (and `where` command) supports boolean logic and comparison operators.

```ebnf
search_expr = or_expr ;
or_expr     = and_expr { "OR" and_expr } ;
and_expr    = unary_expr { [ "AND" ] unary_expr } ;
unary_expr  = [ "NOT" | "!" ] primary_expr ;
primary_expr = "(" search_expr ")"
             | field_cmp
             | free_text
             | subquery ;

field_cmp   = identifier cmp_op value ;
cmp_op      = "=" | "!=" | ">" | ">=" | "<" | "<=" | "IN" | "LIKE" | "MATCHES" ;
free_text   = string_literal | identifier ;
subquery    = "[" [ "search" ] query "]" ;
```

## Commands

OQL supports a variety of commands for data transformation and aggregation.

### `where`
Filters rows based on a search expression.
```
where <search_expr>
```

### `stats`
Calculates aggregations, optionally grouped by fields.
```
stats <agg_func>(<field>) [as <alias>] [, ...] [by <field> [, ...]]
```
Supported functions: `count`, `sum`, `avg`, `min`, `max`, `dc` (distinct count), `values`, `list`.

### `eval`
Calculates new fields or modifies existing ones.
```
eval <field> = <expression> [, ...]
```

### `table`
Selects specific fields to display in a tabular format.
```
table <field> [, ...]
```

### `sort`
Sorts results by one or more fields.
```
sort [+/-] <field> [, ...]
```

### `head` / `tail`
Returns the first or last N results.
```
head <count>
tail <count>
```

### `dedup`
Removes duplicate results based on field values.
```
dedup [count] <field> [, ...]
```

### `rename`
Renames fields.
```
rename <old_field> as <new_field> [, ...]
```

### `fields`
Keeps or removes specific fields.
```
fields [+/-] <field> [, ...]
```

### `lookup`
Enriches data from a lookup table.
```
lookup <table_name> <match_field> [as <alias>] [output <fields>]
```

### `join`
Joins results with a subquery.
```
join [type=left|inner] <field> [search <subquery>]
```

## Computational Cost Modeling

OQL enforces strict limits on query execution to prevent resource exhaustion.

- **MAX_SCAN_BYTES**: Maximum data scanned per query.
- **MAX_WALL_TIME**: Maximum execution time.
- **MAX_MEMORY_BYTES**: Maximum memory usage for aggregations and joins.
- **MAX_ROWS_OUTPUT**: Maximum number of rows returned to the UI.

Queries exceeding these limits are automatically terminated with a `BudgetViolation` error.
