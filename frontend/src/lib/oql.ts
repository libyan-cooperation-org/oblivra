/**
 * OBLIVRA — Oblivra Query Language (OQL) Parser
 * A lightweight grammar for SIEM telemetry filtering.
 * 
 * Syntax: host="node-01" AND severity="high" | top 10
 */

export interface OQLQuery {
  filters: Array<{ field: string; operator: string; value: string }>;
  limit?: number;
  sort?: { field: string; order: 'asc' | 'desc' };
}

export function parseOQL(query: string): OQLQuery {
  const result: OQLQuery = { filters: [] };
  
  // Basic tokenization (naive implementation for demonstration)
  const parts = query.split('|').map(s => s.trim());
  const filterPart = parts[0];
  const commandPart = parts[1];

  // Parse filters: field="value"
  const filterRegex = /(\w+)\s*(=|!=|~)\s*["']([^"']+)["']/g;
  let match;
  while ((match = filterRegex.exec(filterPart)) !== null) {
    result.filters.push({
      field: match[1],
      operator: match[2],
      value: match[3]
    });
  }

  // Parse commands: top 10 message
  if (commandPart) {
    const limitMatch = commandPart.match(/limit\s+(\d+)/i) || commandPart.match(/top\s+(\d+)/i);
    if (limitMatch) {
      result.limit = parseInt(limitMatch[1], 10);
    }
  }

  return result;
}

export function evaluateOQL(query: string, data: any[]): any[] {
  const parsed = parseOQL(query);
  if (parsed.filters.length === 0) return data;

  return data.filter(item => {
    return parsed.filters.every(filter => {
      const itemValue = String(item[filter.field] || '').toLowerCase();
      const filterValue = filter.value.toLowerCase();
      
      switch (filter.operator) {
        case '=': return itemValue === filterValue;
        case '!=': return itemValue !== filterValue;
        case '~': return itemValue.includes(filterValue);
        default: return true;
      }
    });
  }).slice(0, parsed.limit || data.length);
}
