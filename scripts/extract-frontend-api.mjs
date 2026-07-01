#!/usr/bin/env node
/**
 * Extract API Contract from Frontend Source Files
 *
 * Parses API facade files and Zod schema files to build a structured
 * JSON representation of the complete REST API contract.
 *
 * Usage: node scripts/extract-api-contract.mjs > api-contract.json
 */

import { readFileSync, writeFileSync, globSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SRC = resolve(__dirname, '..', 'src');

// ---------------------------------------------------------------------------
// 1. Parse API Facade files → endpoint list
// ---------------------------------------------------------------------------

/**
 * Extract endpoint definitions from an API facade source file.
 * Matches patterns like:
 *   transport.post<T>("/path", input)
 *   transport.get<T>("/path")
 *   transport.patch<T>("/path", input)
 *   transport.put<T>("/path", input)
 *   transport.delete<T>("/path")
 *   transport.stream<T>("/path", input, handlers)
 */
/**
 * Normalize a template literal path like `/visits/${body.sessionId}/messages`
 * to a canonical form like `/api/visits/:sessionId/messages`
 */
function normalizePath(rawPath, domain) {
  // Replace ${body.xxx}, ${input.xxx}, ${query.xxx}, ${xxx} with :xxx
  let normalized = rawPath.replace(/\$\{(?:body|input|query)\.(\w+)\}/g, ':$1');
  normalized = normalized.replace(/\$\{(\w+)\}/g, ':$1');

  // Clean trailing slash
  normalized = normalized.replace(/\/$/, '');

  // Add /api prefix for patient endpoints, /admin is its own prefix
  if (!normalized.startsWith('/admin') && !normalized.startsWith('/api')) {
    normalized = '/api' + normalized;
  }

  return normalized;
}

function parseApiFacade(source, filePath) {
  const endpoints = [];

  /**
   * Preprocess: collapse multi-line method calls into single lines
   * so regex can match across line boundaries.
   */
  const collapsed = source.replace(/\n\s*/g, ' ');

  const allMatches = [];

  // Match: .method<anything...>("path" or `/path`)
  // [^(]* skips the type parameter (including nested generics) until the '('
  const callPattern = /\.(post|get|patch|put|delete|stream)\b[^(]*\(\s*["'`]([\/][^"'`]+)["'`]/g;

  let m;
  while ((m = callPattern.exec(collapsed)) !== null) {
    allMatches.push({
      method: m[1],
      responseType: 'see schemas',
      rawPath: m[2],
    });
  }

  for (const m of allMatches) {
    const httpMethod = m.method.toUpperCase();
    const responseType = m.responseType;
    const rawPath = m.rawPath;

    const domain = inferDomain(filePath);
    const isSSE = httpMethod === 'STREAM';

    // STREAM is always POST over HTTP for SSE
    const method = isSSE ? 'POST' : httpMethod;

    endpoints.push({
      method,
      path: normalizePath(rawPath, domain),
      rawPath,
      responseType,
      domain,
      file: filePath,
      isSSE,
    });
  }

  return endpoints;
}

function inferDomain(filePath) {
  if (filePath.includes('/admin/')) return 'admin';
  if (filePath.includes('/auth/')) return 'auth';
  if (filePath.includes('/patient/')) return 'patient';
  if (filePath.includes('/visits/')) return 'visits';
  if (filePath.includes('/workbench/')) return 'workbench';
  if (filePath.includes('/billing/')) return 'billing';
  if (filePath.includes('/medical-orders/')) return 'medical-orders';
  return 'unknown';
}

// ---------------------------------------------------------------------------
// 2. Parse Zod Schema files → field definitions
// ---------------------------------------------------------------------------

/**
 * Parse Zod schema definitions from a TypeScript source file.
 * Extracts object schemas and their field definitions.
 */
function parseZodSchemas(source, filePath) {
  const schemas = {};

  // Find exported schema constants: export const xxxSchema = z.object({...})
  const schemaPattern = /export\s+const\s+(\w+)\s*=\s*z\.object\(\{([^}]*(?:\{[^}]*\}[^}]*)*)\}\)/gs;

  let match;
  while ((match = schemaPattern.exec(source)) !== null) {
    const name = match[1];
    const body = match[2];
    const fields = parseObjectFields(body);
    if (fields.length > 0) {
      schemas[name] = { name, fields, file: filePath };
    }
  }

  // Also parse z.intersection, z.discriminatedUnion, etc.
  const discriminatedPattern = /export\s+const\s+(\w+)\s*=\s*z\.discriminatedUnion\s*\(\s*["'`](\w+)["'`]/g;
  while ((match = discriminatedPattern.exec(source)) !== null) {
    schemas[match[1]] = { name: match[1], kind: 'discriminatedUnion', discriminator: match[2], file: filePath };
  }

  return schemas;
}

/**
 * Parse fields from a Zod object body string.
 * Matches patterns like: fieldName: z.string().min(1),
 */
function parseObjectFields(body) {
  const fields = [];

  // Split by field definitions (comma-separated at top level)
  // This is a simplified parser - handles common patterns
  const fieldPattern = /(\w+)\s*:\s*(z\.\w+\([^)]*\)(?:\.\w+\([^)]*\))*)/g;

  let match;
  while ((match = fieldPattern.exec(body)) !== null) {
    const name = match[1];
    const typeExpr = match[2];

    fields.push({
      name,
      zodType: typeExpr,
      type: zodToJsonType(typeExpr),
      required: !typeExpr.includes('.optional()') && !typeExpr.includes('.nullable()'),
      constraints: extractConstraints(typeExpr),
    });
  }

  return fields;
}

/**
 * Map Zod type expression to JSON schema type.
 */
function zodToJsonType(expr) {
  if (expr.startsWith('z.string()')) return 'string';
  if (expr.startsWith('z.number()')) return 'number';
  if (expr.startsWith('z.boolean()')) return 'boolean';
  if (expr.startsWith('z.array(')) return 'array';
  if (expr.startsWith('z.object(')) return 'object';
  if (expr.startsWith('z.enum(')) return 'string (enum)';
  if (expr.startsWith('z.union(')) return 'union';
  if (expr.startsWith('z.discriminatedUnion(')) return 'discriminatedUnion';
  if (expr.startsWith('z.literal(')) return 'string (literal)';
  if (expr.startsWith('z.date(') || expr.includes('datetime()')) return 'string (datetime)';
  if (expr.startsWith('z.instanceof(Date)')) return 'string (date)';
  return 'unknown';
}

/**
 * Extract constraints from Zod type expression.
 */
function extractConstraints(expr) {
  const constraints = [];
  const minMatch = expr.match(/\.min\((\d+)\)/);
  const maxMatch = expr.match(/\.max\((\d+)\)/);
  const regexMatch = expr.match(/\.regex\(([^)]+)\)/);
  const trimMatch = expr.includes('.trim()');

  if (minMatch) constraints.push(`min:${minMatch[1]}`);
  if (maxMatch) constraints.push(`max:${maxMatch[1]}`);
  if (regexMatch) constraints.push(`regex:${regexMatch[1]}`);
  if (trimMatch) constraints.push('trimmed');

  return constraints.join(', ');
}

// ---------------------------------------------------------------------------
// 3. Parse TypeScript interfaces/types → response shapes
// ---------------------------------------------------------------------------

function parseTypes(source, filePath) {
  const types = {};

  // Parse interfaces
  const ifacePattern = /export\s+interface\s+(\w+)\s*\{([^}]+)\}/gs;
  let match;
  while ((match = ifacePattern.exec(source)) !== null) {
    const name = match[1];
    const body = match[2];
    const fields = parseInterfaceFields(body);
    if (fields.length > 0) {
      types[name] = { kind: 'interface', name, fields, file: filePath };
    }
  }

  // Parse type aliases (including intersections)
  const typePattern = /export\s+type\s+(\w+)\s*=\s*(.+?)(?:;|\n\s*\n)/gs;
  while ((match = typePattern.exec(source)) !== null) {
    const name = match[1];
    const definition = match[2].trim();
    if (!types[name]) {
      types[name] = { kind: 'type', name, definition, file: filePath };
    }
  }

  return types;
}

function parseInterfaceFields(body) {
  const fields = [];
  const lines = body.split('\n');

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('/**') || trimmed.startsWith('*') || trimmed.startsWith('//')) continue;

    // Match: fieldName?: type; or fieldName: type;
    const fieldMatch = trimmed.match(/^(\w+)(\?)?\s*:\s*(.+?)[,;]?\s*$/);
    if (fieldMatch) {
      fields.push({
        name: fieldMatch[1],
        type: fieldMatch[3].replace(/;$/, '').trim(),
        required: !fieldMatch[2],
      });
    }
  }

  return fields;
}

// ---------------------------------------------------------------------------
// 4. Parse Enum definitions
// ---------------------------------------------------------------------------

function parseEnums(source, filePath) {
  const enums = {};

  // Zod enums: z.enum(["val1", "val2"])
  const zodEnumPattern = /(\w+)\s*=\s*z\.enum\(\s*\[([^\]]+)\]\s*\)/g;
  let match;
  while ((match = zodEnumPattern.exec(source)) !== null) {
    const name = match[1];
    const values = match[2]
      .split(',')
      .map(v => v.trim().replace(/["'`]/g, ''))
      .filter(Boolean);
    enums[name] = { name, values, file: filePath };
  }

  return enums;
}

// ---------------------------------------------------------------------------
// 5. Main extraction
// ---------------------------------------------------------------------------

function main() {
  // Helper: glob that returns [] on error
  const glob = (pattern) => {
    try { return globSync(pattern, { cwd: SRC }); } catch { return []; }
  };

  // Find all API facade files (relative to SRC)
  const apiFiles = [
    SRC + '/features/auth/api/auth-api.ts',
    SRC + '/features/patient/api/index.ts',
    SRC + '/features/visits/api/index.ts',
    SRC + '/features/workbench/api/index.ts',
    SRC + '/features/billing/api/index.ts',
    SRC + '/features/medical-orders/api/index.ts',
    SRC + '/features/admin/api/admin-api.ts',
  ].filter(f => { try { readFileSync(f); return true; } catch { return false; } });

  // Find all schema files
  const schemaGlob = glob('features/*/api/schemas.ts');
  const schemaFiles = [
    ...schemaGlob.map(f => `${SRC}/${f}`),
    `${SRC}/features/workbench/api/timeline-schemas.ts`,
    `${SRC}/lib/api/types.ts`,
  ].filter(f => { try { readFileSync(f); return true; } catch { return false; } });

  // Find all type files
  const typeGlob = glob('features/*/api/types.ts');
  const typeFiles = [
    ...typeGlob.map(f => `${SRC}/${f}`),
    `${SRC}/features/workbench/api/timeline-types.ts`,
  ].filter(f => { try { readFileSync(f); return true; } catch { return false; } });

  // Extract endpoints
  const allEndpoints = [];
  for (const file of apiFiles) {
    try {
      const source = readFileSync(file, 'utf-8');
      const endpoints = parseApiFacade(source, file);
      allEndpoints.push(...endpoints);
      console.error(`[INFO] ${file}: ${endpoints.length} endpoints`);
    } catch (err) {
      console.error(`[WARN] Cannot read ${file}: ${err.message}`);
    }
  }

  // Extract schemas
  const allSchemas = {};
  for (const file of schemaFiles) {
    try {
      const source = readFileSync(file, 'utf-8');
      Object.assign(allSchemas, parseZodSchemas(source, file));
    } catch (err) {
      console.error(`[WARN] Cannot read ${file}: ${err.message}`);
    }
  }

  // Extract types
  const allTypes = {};
  for (const file of typeFiles) {
    try {
      const source = readFileSync(file, 'utf-8');
      Object.assign(allTypes, parseTypes(source, file));
    } catch (err) {
      console.error(`[WARN] Cannot read ${file}: ${err.message}`);
    }
  }

  // Extract enums from lib/api/types.ts
  const allEnums = {};
  try {
    const sharedTypes = readFileSync(`${SRC}/lib/api/types.ts`, 'utf-8');
    Object.assign(allEnums, parseEnums(sharedTypes, `${SRC}/lib/api/types.ts`));
  } catch (err) {
    console.error(`[WARN] Cannot read shared types: ${err.message}`);
  }

  // Try to match endpoints with their request/response schemas
  const enrichedEndpoints = allEndpoints.map(ep => {
    // Try to find request schema by convention
    const requestSchemaNames = findRelatedSchemas(ep, allSchemas, 'input');
    const responseSchemaNames = findRelatedTypes(ep, allTypes, allSchemas);

    return {
      ...ep,
      requestSchemas: requestSchemaNames,
      responseTypes: responseSchemaNames,
    };
  });

  const contract = {
    source: 'frontend-zod-schemas',
    extractedAt: new Date().toISOString(),
    totalEndpoints: enrichedEndpoints.length,
    endpoints: enrichedEndpoints,
    schemas: allSchemas,
    types: allTypes,
    enums: allEnums,
  };

  writeFileSync(
    resolve(__dirname, '..', 'api-contract.json'),
    JSON.stringify(contract, null, 2)
  );

  console.error(`\n[DONE] Extracted ${enrichedEndpoints.length} endpoints to api-contract.json`);
  console.error(`[STATS] Schemas: ${Object.keys(allSchemas).length}, Types: ${Object.keys(allTypes).length}, Enums: ${Object.keys(allEnums).length}`);

  // Also print to stdout for piping
  console.log(JSON.stringify(contract, null, 2));
}

function findRelatedSchemas(endpoint, schemas, suffix) {
  // Look for schema names that match the endpoint path or domain
  const domain = endpoint.domain;
  const pathParts = endpoint.rawPath.split('/').filter(Boolean);
  const matches = [];

  for (const [name, schema] of Object.entries(schemas)) {
    const lower = name.toLowerCase();
    if (lower.includes(domain) || pathParts.some(p => lower.includes(p.toLowerCase()))) {
      matches.push(name);
    }
  }

  return matches;
}

function findRelatedTypes(endpoint, types, schemas) {
  const responseType = endpoint.responseType;
  if (responseType === 'void' || responseType === 'void;') return [];

  const matches = [];
  for (const [name, type] of Object.entries(types)) {
    if (name === responseType || responseType.includes(name)) {
      matches.push(name);
    }
  }

  return matches;
}

main();
