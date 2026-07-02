#!/usr/bin/env node
/**
 * Request body & query parameter drift detection.
 *
 * Compares frontend input schemas (Zod) against backend request structs (Go)
 * and query parameter definitions.
 *
 * Inputs:
 *   frontend-fields.json (from ../neuhis-agent-front/frontend-fields.json)
 *   backend-fields.json  (from ./backend-fields.json)
 *   api-contract.json    (from ../neuhis-agent-front/api-contract.json)
 *
 * Output: drift-report-request.json
 */
import { readFileSync, writeFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');
const FE_DIR = resolve(ROOT, '..', 'neuhis-agent-front');

// ====== Load Data ======

function loadJSON(path) {
  try { return JSON.parse(readFileSync(path, 'utf-8')); } catch { return {}; }
}

const apiContract = loadJSON(resolve(FE_DIR, 'api-contract.json'));
const frontendFields = loadJSON(resolve(FE_DIR, 'frontend-fields.json'));
const backendFields = loadJSON(resolve(ROOT, 'backend-fields.json'));

const ZOD_SCHEMAS = frontendFields.schemas || {};
const GO_STRUCTS = backendFields.structs || {};

// ====== Helpers ======

function normPath(p) {
  return p.replace(/:\w+/g, ':param').replace(/\/$/, '');
}

// ====== Zod → Go Type Compatibility (same as compare-fields.mjs) ======

function normalizeGoType(goType) {
  let t = goType.trim();
  t = t.replace(/^\*/, '');
  t = t.replace(/^\[\]/, '');
  t = t.replace(/^model\./, '');
  return t;
}

const ZOD_TO_GO_COMPATIBLE = {
  'string': new Set(['string']),
  'number': new Set(['int', 'int8', 'int16', 'int32', 'int64', 'uint', 'uint8', 'uint16', 'uint32', 'uint64', 'float32', 'float64']),
  'boolean': new Set(['bool']),
  'datetime': new Set(['time.Time']),
};

function checkTypeCompatibility(zodType, goType) {
  if (!zodType || !goType) return true;
  const baseGo = normalizeGoType(goType);
  if (zodType === 'enum' || zodType === 'literal' || zodType === 'schemaRef' || zodType === 'unknown') return true;
  if (zodType === 'string') return baseGo === 'string';
  if (zodType === 'number') return ZOD_TO_GO_COMPATIBLE.number.has(baseGo);
  if (zodType === 'boolean') return baseGo === 'bool';
  if (zodType === 'datetime') return baseGo === 'time.Time';
  if (zodType === 'array') return goType.trim().startsWith('[]');
  return true;
}

// ====== Main Comparison ======

function main() {
  const driftItems = [];
  let requestsCompared = 0;
  let queryEndpointsChecked = 0;

  const feEndpoints = frontendFields.endpoints || [];
  const beEndpoints = backendFields.endpoints || [];

  // Build backend endpoint index by normalized signature
  const beBySig = {};
  for (const ep of beEndpoints) {
    const sig = `${ep.method} ${normPath(ep.path)}`;
    beBySig[sig] = ep;
  }

  // ======== 1. Request Body Comparison ========

  for (const feEp of feEndpoints) {
    const sig = `${feEp.httpMethod} ${normPath(feEp.path)}`;
    const beEp = beBySig[sig];

    // Skip SSE and GET endpoints (no request body)
    if (feEp.isSSE) continue;
    if (feEp.httpMethod === 'GET') continue;
    if (feEp.httpMethod === 'DELETE') continue;

    const feReqSchemaName = feEp.requestSchema;
    const feReqFields = feEp.requestFields || [];

    if (!beEp) {
      if (feReqSchemaName && feReqFields.length > 0) {
        driftItems.push({
          severity: 'MEDIUM',
          category: 'request_endpoint_missing',
          endpoint: sig,
          field: '(endpoint)',
          expected: `${feEp.httpMethod} ${feEp.path} — has request schema "${feReqSchemaName}"`,
          actual: 'Endpoint NOT FOUND in backend',
          description: `Frontend defines request body for ${sig} but backend endpoint not found.`,
          fixHint: 'Add backend endpoint or remove from frontend.',
        });
      }
      continue;
    }

    const beReqType = beEp.requestType;
    const beReqFields = beEp.requestFields || [];

    // Both sides have no request body
    if (feReqFields.length === 0 && beReqFields.length === 0) continue;

    // Frontend has request body, backend doesn't
    if (feReqFields.length > 0 && beReqFields.length === 0) {
      requestsCompared++;
      for (const feF of feReqFields) {
        driftItems.push({
          severity: 'HIGH',
          category: 'request_body_missing_backend',
          endpoint: sig,
          field: feF.name,
          expected: `Zod: ${feF.name} (${feF.zodType}, required=${feF.required}) in "${feReqSchemaName}"`,
          actual: 'No request struct found in backend',
          description: `Frontend expects "${feF.name}" in request body but backend has no request type for ${sig}.`,
          fixHint: `Add request struct or BindJSON type for ${sig}.`,
        });
      }
      continue;
    }

    // Backend has request body, frontend doesn't
    if (feReqFields.length === 0 && beReqFields.length > 0) {
      requestsCompared++;
      for (const beF of beReqFields) {
        if (beF.required) {
          driftItems.push({
            severity: 'MEDIUM',
            category: 'request_body_extra_backend',
            endpoint: sig,
            field: beF.jsonName,
            expected: 'NOT in frontend input schema',
            actual: `Go: ${beF.goType} (required) in "${beReqType}"`,
            description: `Backend requires "${beF.jsonName}" but frontend has no input schema for ${sig}.`,
            fixHint: `Add "${beF.jsonName}" to frontend input schema or make optional.`,
          });
        }
      }
      continue;
    }

    // Both have request fields — compare them
    requestsCompared++;

    const feFieldMap = {};
    for (const f of feReqFields) feFieldMap[f.name] = f;

    const beFieldMap = {};
    for (const f of beReqFields) beFieldMap[f.jsonName] = f;

    const allNames = new Set([...Object.keys(feFieldMap), ...Object.keys(beFieldMap)]);

    for (const fieldName of allNames) {
      const feF = feFieldMap[fieldName];
      const beF = beFieldMap[fieldName];

      // Field in frontend but not backend
      if (feF && !beF) {
        driftItems.push({
          severity: 'HIGH',
          category: 'request_field_missing',
          endpoint: sig,
          field: fieldName,
          expected: `Zod: ${fieldName} (${feF.zodType}, required=${feF.required})`,
          actual: 'NOT IN Go request struct',
          description: `Request field "${fieldName}" defined in frontend "${feReqSchemaName}" but missing from Go "${beReqType}".`,
          fixHint: `Add "${fieldName}" to ${beReqType} struct.`,
        });
        continue;
      }

      // Field in backend but not frontend (and is required)
      if (!feF && beF && beF.required) {
        driftItems.push({
          severity: 'MEDIUM',
          category: 'request_field_extra',
          endpoint: sig,
          field: fieldName,
          expected: 'NOT in frontend input schema',
          actual: `Go: ${beF.goType} (required)`,
          description: `Backend requires "${fieldName}" but not in frontend input schema "${feReqSchemaName}".`,
          fixHint: `Add "${fieldName}" to frontend schema or make optional in Go.`,
        });
        continue;
      }

      // Both exist — check type compatibility
      if (feF && beF) {
        if (!checkTypeCompatibility(feF.zodType, beF.goType)) {
          driftItems.push({
            severity: 'HIGH',
            category: 'request_type_mismatch',
            endpoint: sig,
            field: fieldName,
            expected: `Zod: ${feF.zodType}`,
            actual: `Go: ${beF.goType}`,
            description: `Request field "${fieldName}" type mismatch: Zod ${feF.zodType} vs Go ${beF.goType}.`,
            fixHint: `Align Go type of "${fieldName}" with frontend Zod ${feF.zodType}.`,
          });
        }

        // Check constraint parity
        const feCons = feF.constraints || {};
        const beCons = beF.constraints || {};

        if (feCons.min !== undefined && beCons.min === undefined && beCons.gt === undefined) {
          driftItems.push({
            severity: 'MEDIUM',
            category: 'request_constraint_missing',
            endpoint: sig,
            field: fieldName,
            expected: `Zod: .min(${feCons.min})`,
            actual: 'Go: no min/gt constraint',
            description: `Request field "${fieldName}": Zod .min(${feCons.min}) not enforced in Go.`,
            fixHint: `Add binding:"min=${feCons.min}" to "${fieldName}" in ${beReqType}.`,
          });
        }

        if (feCons.max !== undefined && beCons.max === undefined) {
          driftItems.push({
            severity: 'MEDIUM',
            category: 'request_constraint_missing',
            endpoint: sig,
            field: fieldName,
            expected: `Zod: .max(${feCons.max})`,
            actual: 'Go: no max constraint',
            description: `Request field "${fieldName}": Zod .max(${feCons.max}) not enforced in Go.`,
            fixHint: `Add binding:"max=${feCons.max}" to "${fieldName}" in ${beReqType}.`,
          });
        }
      }
    }
  }

  // ======== 2. Query Parameter Comparison ========

  for (const feEp of feEndpoints) {
    if (feEp.httpMethod !== 'GET') continue;

    const sig = `${feEp.httpMethod} ${normPath(feEp.path)}`;
    const beEp = beBySig[sig];
    if (!beEp) continue;

    const feReqFields = feEp.requestFields || [];
    if (feReqFields.length === 0) continue; // No query params defined in frontend

    queryEndpointsChecked++;

    // Backend query params extracted from handler code
    const beQueryParams = beEp.queryParams || [];

    const feParamNames = new Set(feReqFields.map(f => f.name));
    const beParamNames = new Set(beQueryParams);

    // Frontend query params missing in backend
    for (const feF of feReqFields) {
      if (!beParamNames.has(feF.name) && feF.required) {
        driftItems.push({
          severity: 'HIGH',
          category: 'query_param_missing',
          endpoint: sig,
          field: feF.name,
          expected: `Zod: required query param "${feF.name}" (${feF.zodType})`,
          actual: `Backend query params: [${beQueryParams.join(', ')}]`,
          description: `Required query param "${feF.name}" in frontend but not extracted by backend handler.`,
          fixHint: `Add c.Query("${feF.name}") to handler for ${sig}.`,
        });
      } else if (!beParamNames.has(feF.name) && !feF.required) {
        driftItems.push({
          severity: 'LOW',
          category: 'query_param_missing_optional',
          endpoint: sig,
          field: feF.name,
          expected: `Zod: optional query param "${feF.name}"`,
          actual: 'NOT in backend query extraction',
          description: `Optional query param "${feF.name}" in frontend but not found in backend handler.`,
          fixHint: `Add c.Query("${feF.name}") to handler or document as unsupported.`,
        });
      }
    }

    // Backend query params not in frontend
    for (const bp of beQueryParams) {
      if (!feParamNames.has(bp) && !['page', 'pageSize', 'cursor'].includes(bp)) {
        driftItems.push({
          severity: 'LOW',
          category: 'query_param_extra',
          endpoint: sig,
          field: bp,
          expected: 'NOT in frontend query schema',
          actual: `Backend reads "${bp}" from query string`,
          description: `Backend reads "${bp}" query param but frontend schema doesn't define it.`,
          fixHint: `Add "${bp}" to frontend query schema or remove from backend.`,
        });
      }
    }
  }

  // ======== 3. HTTP Method Consistency ========

  // Check that SSE endpoints are POST (with stream transport)
  for (const feEp of feEndpoints) {
    if (!feEp.isSSE) continue;
    const sig = `${feEp.httpMethod} ${normPath(feEp.path)}`;
    const beEp = beBySig[sig];
    if (beEp && beEp.method !== 'POST') {
      driftItems.push({
        severity: 'HIGH',
        category: 'sse_method_mismatch',
        endpoint: sig,
        field: 'HTTP method',
        expected: 'POST (SSE streaming)',
        actual: `${beEp.method}`,
        description: `SSE endpoint ${sig} should use POST but backend has ${beEp.method}.`,
        fixHint: `Change backend route method to POST.`,
      });
    }
  }

  // Check that 204 endpoints are correctly typed
  for (const beEp of beEndpoints) {
    if (beEp.responseType === 'void') {
      const sig = `${beEp.method} ${normPath(beEp.path)}`;
      const feEp = feEndpoints.find(e => `${e.httpMethod} ${normPath(e.path)}` === sig);
      if (feEp && feEp.responseSchema && feEp.responseSchema !== 'unknown') {
        driftItems.push({
          severity: 'MEDIUM',
          category: 'void_response_mismatch',
          endpoint: sig,
          field: 'response type',
          expected: `Backend returns 204 (no body)`,
          actual: `Frontend expects schema "${feEp.responseSchema}"`,
          description: `Backend returns 204 for ${sig} but frontend expects a response body.`,
          fixHint: `Align response expectations: either add body or update frontend.`,
        });
      }
    }
  }

  // ======== Summary ========

  const bySeverity = {};
  const byCategory = {};
  for (const item of driftItems) {
    bySeverity[item.severity] = (bySeverity[item.severity] || 0) + 1;
    byCategory[item.category] = (byCategory[item.category] || 0) + 1;
  }

  const report = {
    comparedAt: new Date().toISOString(),
    requestsCompared,
    queryEndpointsChecked,
    totalDriftItems: driftItems.length,
    bySeverity,
    byCategory,
    items: driftItems,
  };

  writeFileSync(resolve(ROOT, 'drift-report-request.json'), JSON.stringify(report, null, 2));

  console.error('=== Request Body & Query Parameter Drift Report ===');
  console.error(`Requests compared:      ${requestsCompared}`);
  console.error(`Query endpoints checked: ${queryEndpointsChecked}`);
  console.error(`Total drift items:      ${driftItems.length}`);
  console.error('');
  console.error('By Severity:');
  for (const [sev, count] of Object.entries(bySeverity).sort()) {
    console.error(`  ${sev}: ${count}`);
  }
  console.error('');
  console.error('By Category:');
  for (const [cat, count] of Object.entries(byCategory).sort()) {
    console.error(`  ${cat}: ${count}`);
  }

  if (driftItems.length > 0) {
    console.error('\n--- Drift Items ---');
    for (const item of driftItems) {
      console.error(`[${item.severity}] ${item.category}: ${item.field} @ ${item.endpoint}`);
      console.error(`       ${item.description}`);
    }
  } else {
    console.error('\n✅ No request drift detected!');
  }

  console.log(JSON.stringify(report, null, 2));
}

main();
