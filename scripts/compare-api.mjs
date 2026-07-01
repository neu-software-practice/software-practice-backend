#!/usr/bin/env node
/**
 * Compare frontend API contract vs backend implementation to find drift.
 *
 * Usage: node scripts/compare-api.mjs [frontend.json] [backend.json]
 * Output: drift-report.json
 */

import { readFileSync, writeFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');

const frontendFile = process.argv[2] || resolve(ROOT, '..', 'neuhis-agent-front', 'api-contract.json');
const backendFile = process.argv[3] || resolve(ROOT, 'backend-api.json');

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Normalize a path by replacing parameter names with placeholders */
function normalizePath(path) {
  // Replace :paramName and ${xxx} with :pN
  let normalized = path.replace(/:\w+/g, ':param');
  // Remove trailing slash
  normalized = normalized.replace(/\/$/, '');
  return normalized;
}

/** Extract parameter names from a path */
function extractParams(path) {
  const params = [];
  const matches = path.matchAll(/:(\w+)/g);
  for (const m of matches) {
    params.push(m[1]);
  }
  return params;
}

/** Create a signature for endpoint matching: METHOD + normalized path */
function signature(ep) {
  return `${ep.method} ${normalizePath(ep.path || ep.fullPath)}`;
}

// ---------------------------------------------------------------------------
// Main comparison
// ---------------------------------------------------------------------------

function main() {
  const frontend = JSON.parse(readFileSync(frontendFile, 'utf-8'));
  const backend = JSON.parse(readFileSync(backendFile, 'utf-8'));

  const frontendEndpoints = frontend.endpoints || [];
  const backendEndpoints = backend.endpoints || [];

  const driftItems = [];

  // Build index by normalized signature
  const feBySig = {};
  for (const ep of frontendEndpoints) {
    const sig = signature(ep);
    if (!feBySig[sig]) feBySig[sig] = [];
    feBySig[sig].push(ep);
  }

  const beBySig = {};
  for (const ep of backendEndpoints) {
    const sig = signature(ep);
    if (!beBySig[sig]) beBySig[sig] = [];
    beBySig[sig].push(ep);
  }

  const feSigs = new Set(Object.keys(feBySig));
  const beSigs = new Set(Object.keys(beBySig));

  // ─── 1. Missing endpoints (frontend has it, backend doesn't) ───
  for (const sig of feSigs) {
    if (!beSigs.has(sig)) {
      const eps = feBySig[sig];
      for (const ep of eps) {
        driftItems.push({
          severity: 'CRITICAL',
          category: 'missing_endpoint',
          endpoint: sig,
          expected: `${ep.method} ${ep.path} — defined in frontend`,
          actual: 'NOT FOUND in backend',
          description: `Endpoint ${ep.method} ${ep.path} is defined in frontend but missing from backend routes`,
          file: 'internal/handler/router.go',
          fixHint: `Add route: auth.${ep.method}("${ep.rawPath}", router.Xxx.Yyy)`,
        });
      }
    }
  }

  // ─── 2. Extra endpoints (backend has it, frontend doesn't) ───
  for (const sig of beSigs) {
    if (!feSigs.has(sig)) {
      const eps = beBySig[sig];
      for (const ep of eps) {
        // Health check is acceptable extra
        if (ep.fullPath === '/api/health') {
          driftItems.push({
            severity: 'LOW',
            category: 'extra_endpoint',
            endpoint: sig,
            expected: 'Not defined in frontend contract',
            actual: `${ep.method} ${ep.fullPath}`,
            description: `/api/health is a standard health check endpoint. Not in frontend contract but acceptable.`,
            file: 'internal/handler/router.go',
            fixHint: 'No fix needed — health check is standard infrastructure',
          });
        } else {
          driftItems.push({
            severity: 'HIGH',
            category: 'extra_endpoint',
            endpoint: sig,
            expected: 'Not defined in frontend contract',
            actual: `${ep.method} ${ep.fullPath}`,
            description: `Endpoint ${ep.method} ${ep.fullPath} exists in backend but not in frontend contract. Check if needed.`,
            file: 'internal/handler/router.go',
            fixHint: 'Verify if this endpoint should exist. Remove or add to frontend contract.',
          });
        }
      }
    }
  }

  // ─── 3. Path parameter name mismatches ───
  for (const sig of feSigs) {
    if (!beSigs.has(sig)) continue; // Already reported as missing

    const feEps = feBySig[sig];
    const beEps = beBySig[sig];

    for (const feEp of feEps) {
      const feParams = extractParams(feEp.path);
      const fePath = feEp.path;

      for (const beEp of beEps) {
        const beParams = extractParams(beEp.fullPath);
        const bePath = beEp.fullPath;

        // Same number of params
        if (feParams.length === beParams.length && feParams.length > 0) {
          for (let i = 0; i < feParams.length; i++) {
            if (feParams[i] !== beParams[i]) {
              driftItems.push({
                severity: 'MEDIUM',
                category: 'path_param_mismatch',
                endpoint: sig,
                field: feParams[i],
                expected: `Path param named ":${feParams[i]}" (frontend code: ${feEp.rawPath})`,
                actual: `Path param named ":${beParams[i]}" (backend route: ${bePath})`,
                description: `Path parameter name mismatch: frontend uses ":${feParams[i]}" but backend uses ":${beParams[i]}"`,
                file: 'internal/handler/router.go',
                fixHint: `Update backend route param from ":${beParams[i]}" to ":${feParams[i]}"`,
              });
            }
          }
        }
      }
    }
  }

  // ─── Summary ───
  const bySeverity = { CRITICAL: 0, HIGH: 0, MEDIUM: 0, LOW: 0 };
  for (const item of driftItems) {
    bySeverity[item.severity] = (bySeverity[item.severity] || 0) + 1;
  }

  const report = {
    comparedAt: new Date().toISOString(),
    frontendFile,
    backendFile,
    frontendEndpointCount: frontendEndpoints.length,
    backendEndpointCount: backendEndpoints.length,
    totalDriftItems: driftItems.length,
    bySeverity,
    items: driftItems,
    summary: `${driftItems.length} drift items: ${bySeverity.CRITICAL} CRITICAL, ${bySeverity.HIGH} HIGH, ${bySeverity.MEDIUM} MEDIUM, ${bySeverity.LOW} LOW`,
  };

  writeFileSync(resolve(ROOT, 'drift-report.json'), JSON.stringify(report, null, 2));

  // Print summary
  console.error('=== Drift Report ===');
  console.error(`Frontend: ${frontendEndpoints.length} endpoints`);
  console.error(`Backend:  ${backendEndpoints.length} endpoints`);
  console.error(`Drift:    ${driftItems.length} items`);
  console.error(`  CRITICAL: ${bySeverity.CRITICAL}`);
  console.error(`  HIGH:     ${bySeverity.HIGH}`);
  console.error(`  MEDIUM:   ${bySeverity.MEDIUM}`);
  console.error(`  LOW:      ${bySeverity.LOW}`);

  if (driftItems.length > 0) {
    console.error('\n--- Drift Items ---');
    for (const item of driftItems) {
      console.error(`[${item.severity}] ${item.category}: ${item.description}`);
    }
  } else {
    console.error('\n✅ No drift detected!');
  }

  console.log(JSON.stringify(report, null, 2));
}

main();
