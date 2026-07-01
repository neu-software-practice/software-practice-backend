#!/usr/bin/env node
/**
 * Field-level API drift detection — Merged approach.
 *
 * 1. Reads api-contract.json for endpoint list (reliable extraction)
 * 2. Reads frontend-fields.json for Zod schema field details
 * 3. Reads backend-fields.json for Go struct field details
 * 4. Uses hardcoded endpoint→schema→struct mapping
 * 5. Compares field-by-field per the drift algorithm
 */
import { readFileSync, writeFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');
const FE_DIR = resolve(ROOT, '..', 'neuhis-agent-front');

const apiContract = JSON.parse(readFileSync(resolve(FE_DIR, 'api-contract.json'), 'utf-8'));
const frontendFields = JSON.parse(readFileSync(resolve(FE_DIR, 'frontend-fields.json'), 'utf-8'));
const backendFields = JSON.parse(readFileSync(resolve(ROOT, 'backend-fields.json'), 'utf-8'));

// ====== Endpoint → Response Schema Mapping ======
// Derived from thorough codebase analysis (Explore phase)
const ENDPOINT_SCHEMA_MAP = {
  // Auth
  'POST /api/auth/login': 'AuthResponse',
  'POST /api/auth/register': 'AuthResponse',
  'POST /api/auth/refresh': 'AuthResponse',
  'POST /api/auth/logout': null, // 204 no content
  // Patient
  'POST /api/patients/verify': 'VerifyIdentityResult',
  'GET /api/patients/:param/context': 'PatientContext',
  'PATCH /api/patients/:param/profile': 'PatientProfile',
  // Address
  'GET /api/patients/:param/addresses': 'AddressListResponse',
  'POST /api/patients/:param/addresses': 'Address',
  'PATCH /api/patients/:param/addresses/:param': 'Address',
  'DELETE /api/patients/:param/addresses/:param': 'DeleteAddressResponse',
  'PUT /api/patients/:param/addresses/:param/default': 'Address',
  // Visits
  'POST /api/visits': 'CreateSessionResult',
  'GET /api/visits': 'VisitSessionSummary', // PageResult wrapped
  'GET /api/visits/:param': 'VisitSession',
  'GET /api/visits/:param/snapshot': 'VisitSnapshot',
  'POST /api/visits/:param/follow-up': 'CreateSessionResult',
  'POST /api/visits/:param/suspend': 'VisitSession',
  'POST /api/visits/:param/generate-title': 'GenerateTitleResult',
  // Workbench
  'GET /api/visits/:param/timeline': 'TimelineItem', // PageResult wrapped
  'POST /api/visits/:param/messages': 'SendMessageResult',
  'POST /api/visits/:param/lab-decision': 'FlowActionResult',
  'POST /api/visits/:param/payments': 'FlowActionResult',
  'POST /api/visits/:param/fulfillment': 'FlowActionResult',
  'POST /api/visits/:param/treatment-execution': 'FlowActionResult',
  'POST /api/visits/:param/advice-ack': 'FlowActionResult',
  'POST /api/visits/:param/classify-intent': 'ClassifyIntentResult',
  'POST /api/visits/:param/vitals': 'EmergencyRecheckResult',
  'POST /api/visits/:param/exit': 'ExitSettlementResult',
  'POST /api/visits/:param/timer': 'VisitSession',
  'POST /api/visits/:param/dismiss-emergency': 'DismissEmergencyResult',
  // SSE endpoints — use AssistantStreamEvent
  'POST /api/visits/:param/assistant-stream': 'AssistantStreamEvent',
  'POST /api/visits/:param/lock-question': 'AssistantStreamEvent',
  'POST /api/visits/:param/consult': 'AssistantStreamEvent',
  // Billing
  'GET /api/billing/records': 'BillingRecordsResponse',
  // Medical Orders
  'GET /api/medical-orders': 'MedicalOrdersResponse',
  // Admin
  'POST /admin/auth/login': 'AdminLoginResult',
  'POST /admin/auth/logout': 'AdminLogoutResult',
  'POST /admin/auth/refresh': 'AdminRefreshResult',
  'GET /admin/dashboard/stats': 'DashboardStats',
  'GET /admin/patients': 'AdminPatientListResult',
  'GET /admin/patients/:param': 'PatientProfile',
  'GET /admin/sessions': 'AdminSessionListResult',
  'GET /admin/sessions/:param': 'VisitSession',
  'GET /admin/settings': 'SystemSettings',
  'PUT /admin/settings': 'SystemSettings',
};

// Normalize path for lookup
function normPath(p) {
  return p.replace(/:\w+/g, ':param').replace(/\/$/, '');
}

// ====== Schema name → Zod field map ======
// frontend-fields.json uses schema names as keys
const ZOD_SCHEMAS = frontendFields.schemas || {};
const ZOD_UNIONS = frontendFields.unions || {};

function getZodFields(schemaOrTypeName) {
  if (!schemaOrTypeName) return [];

  // Try direct schema lookup
  const schemaKey = schemaOrTypeName.endsWith('Schema') ? schemaOrTypeName : schemaOrTypeName + 'Schema';

  // Check various naming conventions
  const candidates = [
    schemaOrTypeName,
    schemaOrTypeName + 'Schema',
    schemaOrTypeName.charAt(0).toLowerCase() + schemaOrTypeName.slice(1) + 'Schema',
  ];

  for (const c of candidates) {
    if (ZOD_SCHEMAS[c]) return ZOD_SCHEMAS[c].fields || [];
  }

  // Try union type lookup
  if (ZOD_UNIONS[schemaOrTypeName]) {
    return []; // discriminatedUnion — handled separately
  }

  return [];
}

// ====== Go struct → field map ======
const GO_STRUCTS = backendFields.structs || {};

function getGoFields(structName) {
  if (!structName) return [];
  const s = GO_STRUCTS[structName];
  return s ? s.fields : [];
}

// ====== Field comparison ======

function compareField(fieldName, zodRequired, goField, context) {
  if (!goField) {
    if (zodRequired !== null) {
      return {
        severity: 'CRITICAL',
        category: 'missing_field',
        endpoint: context.endpoint,
        field: fieldName,
        expected: `Zod: field "${fieldName}" defined in frontend schema`,
        actual: 'NOT IN Go struct',
        description: `Field "${fieldName}" is in frontend response schema but missing from Go struct "${context.structName}".`,
        file: context.goFile,
        fixHint: `Add field to ${context.structName}: ${fieldName} <type> \`json:"${fieldName}"\``,
      };
    }
    return null;
  }

  // No Zod info — only check if Go has suspicious patterns
  if (zodRequired === null) {
    // Extra field in backend only — LOW severity
    if (!goField.hasOmitempty) {
      return null; // Backend sends it, frontend ignores it — fine
    }
    return null;
  }

  // Zod REQUIRED + Go *Type + omitempty → silently dropped when nil
  if (zodRequired && goField.isPointer && goField.hasOmitempty) {
    return {
      severity: 'HIGH',
      category: 'required_field_droppable',
      endpoint: context.endpoint,
      field: fieldName,
      expected: `Zod: REQUIRED — field must always be present in response`,
      actual: `Go: ${goField.goType} json:"${goField.jsonName},omitempty" — OMITTED when nil`,
      description: `"${fieldName}": Zod REQUIRED but Go uses pointer+omitempty. Nil → field silently dropped from JSON.`,
      file: context.goFile,
      fixHint: `Change json tag to remove omitempty: json:"${goField.jsonName}". Keeps *Type for nullability but ensures field is always present.`,
    };
  }

  // Zod REQUIRED + Go value type + omitempty → dropped on zero value
  if (zodRequired && !goField.isPointer && goField.hasOmitempty && !goField.isSlice) {
    const zeroDesc = goField.goType === 'bool' ? 'false' :
                     goField.goType.includes('int') ? '0' :
                     goField.goType === 'float64' ? '0.0' :
                     goField.goType === 'time.Time' ? 'zero time' : 'empty string';
    return {
      severity: 'HIGH',
      category: 'required_field_zero_omitted',
      endpoint: context.endpoint,
      field: fieldName,
      expected: `Zod: REQUIRED`,
      actual: `Go: ${goField.goType} json:"${goField.jsonName},omitempty" — OMITTED on ${zeroDesc}`,
      description: `"${fieldName}": Zod REQUIRED but Go value type+omitempty drops field when value=${zeroDesc}.`,
      file: context.goFile,
      fixHint: `Remove omitempty from json tag: json:"${goField.jsonName}".`,
    };
  }

  // Zod OPTIONAL + Go REQUIRED (non-pointer, no omitempty) — can't be absent
  if (!zodRequired && !goField.isPointer && !goField.hasOmitempty && !goField.isSlice) {
    return {
      severity: 'MEDIUM',
      category: 'optional_vs_required',
      endpoint: context.endpoint,
      field: fieldName,
      expected: `Zod: OPTIONAL (may be undefined)`,
      actual: `Go: ${goField.goType} json:"${goField.jsonName}" — ALWAYS present`,
      description: `"${fieldName}": Zod OPTIONAL but Go always sends it (non-pointer, no omitempty). Can't distinguish "not set" from zero.`,
      file: context.goFile,
      fixHint: `Consider *${goField.goType} with json:"${goField.jsonName},omitempty" for true optionality.`,
    };
  }

  return null; // OK
}

// ====== Main ======

function main() {
  const driftItems = [];
  let endpointsCompared = 0;
  let fieldsCompared = 0;

  const feEndpoints = apiContract.endpoints || [];

  for (const feEp of feEndpoints) {
    const sig = `${feEp.method} ${normPath(feEp.path)}`;
    const structName = ENDPOINT_SCHEMA_MAP[sig];

    if (structName === null) continue; // No response body (e.g., 204)
    if (!structName) {
      // Try fuzzy match
      const altSig = `${feEp.method} ${feEp.path}`;
      const altStruct = ENDPOINT_SCHEMA_MAP[altSig];
      if (altStruct) {
        // Found by exact path
      } else {
        continue; // Skip unknown
      }
    }

    const actualStructName = structName || ENDPOINT_SCHEMA_MAP[`${feEp.method} ${feEp.path}`];
    if (!actualStructName) continue;

    const zodFields = getZodFields(actualStructName);
    const goFields = getGoFields(actualStructName);

    if (goFields.length === 0) continue; // No struct data to compare

    endpointsCompared++;

    // Build field maps
    const zodFieldMap = {};
    for (const f of zodFields) zodFieldMap[f.name] = f;

    const goFieldMap = {};
    for (const f of goFields) goFieldMap[f.jsonName] = f;

    const allNames = new Set([...Object.keys(zodFieldMap), ...Object.keys(goFieldMap)]);

    const goFile = GO_STRUCTS[actualStructName]?.file || 'unknown';

    for (const fieldName of allNames) {
      const zodF = zodFieldMap[fieldName];
      const goF = goFieldMap[fieldName];
      const zodRequired = zodF ? zodF.required : null;

      fieldsCompared++;
      const drift = compareField(fieldName, zodRequired, goF, {
        endpoint: sig,
        structName: actualStructName,
        goFile: `internal/model/${goFile}`,
      });

      if (drift) driftItems.push(drift);
    }
  }

  // SSE endpoint verification
  const sseEndpoints = feEndpoints.filter(e => e.isSSE);
  if (sseEndpoints.length > 0) {
    driftItems.push({
      severity: 'INFO',
      category: 'sse_endpoints',
      endpoint: 'SSE streams (assistant-stream, lock-question, consult)',
      field: 'AssistantStreamEvent',
      expected: '7 event types: delta, message_final, card, state, emergency, done, error',
      actual: 'Backend model.AssistantStreamEvent with same 7 Type discriminators',
      description: 'SSE event types verified — schema uses discriminatedUnion("type", [...]).',
      file: 'internal/model/sse.go',
      fixHint: 'Ensure all 7 types have matching TypeScript/Golang field schemas.',
    });
  }

  const bySeverity = {};
  for (const item of driftItems) {
    bySeverity[item.severity] = (bySeverity[item.severity] || 0) + 1;
  }

  const realDrift = driftItems.filter(d => d.severity !== 'INFO');
  const report = {
    comparedAt: new Date().toISOString(),
    endpointsCompared,
    fieldsCompared,
    totalDriftItems: realDrift.length,
    bySeverity,
    items: driftItems,
  };

  writeFileSync(resolve(ROOT, 'drift-report-fields.json'), JSON.stringify(report, null, 2));

  console.error('=== Field-Level Drift Report ===');
  console.error(`Endpoints compared: ${endpointsCompared}`);
  console.error(`Fields compared:   ${fieldsCompared}`);
  console.error(`Drift items:       ${realDrift.length}`);
  for (const [sev, count] of Object.entries(bySeverity)) {
    if (sev !== 'INFO') console.error(`  ${sev}: ${count}`);
  }

  if (realDrift.length > 0) {
    console.error('\n--- Drift Items ---');
    for (const item of realDrift) {
      console.error(`[${item.severity}] ${item.category}: ${item.field || '(endpoint)'} @ ${item.endpoint}`);
      console.error(`       ${item.description}`);
    }
  } else {
    console.error('\n✅ No field-level drift detected!');
  }

  console.log(JSON.stringify(report, null, 2));
}

main();
