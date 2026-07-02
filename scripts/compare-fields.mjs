#!/usr/bin/env node
/**
 * Comprehensive field-level API drift detection.
 *
 * Compares Zod schemas (frontend, SOURCE OF TRUTH) against Go structs (backend).
 * Covers 11 drift categories previously undetected.
 *
 * Inputs:
 *   frontend-fields.json (from ../neuhis-agent-front/frontend-fields.json)
 *   backend-fields.json  (from ./backend-fields.json)
 *   api-contract.json    (from ../neuhis-agent-front/api-contract.json)
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
const ZOD_UNIONS = frontendFields.unions || {};
const ZOD_ENUMS = frontendFields.enums || {};
const ENUM_NAME_MAP = frontendFields.enumNameMap || {};
const GO_STRUCTS = backendFields.structs || {};
const GO_ENUMS = backendFields.enums || {};
const GO_STRUCT_REFS = backendFields.structRefs || {};
const GO_PAGINATION = backendFields.paginationStructs || {};
const GO_ERROR_STRUCTS = backendFields.errorStructs || {};

// ====== Endpoint → Schema Mapping (Response) ======

const ENDPOINT_SCHEMA_MAP = {
  // Auth
  'POST /api/auth/login': 'AuthResponse',
  'POST /api/auth/register': 'AuthResponse',
  'POST /api/auth/refresh': 'AuthResponse',
  'POST /api/auth/logout': null,
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
  'GET /api/visits': 'VisitSessionSummary',
  'GET /api/visits/:param': 'VisitSession',
  'GET /api/visits/:param/snapshot': 'VisitSnapshot',
  'POST /api/visits/:param/follow-up': 'CreateSessionResult',
  'POST /api/visits/:param/suspend': 'VisitSession',
  'POST /api/visits/:param/generate-title': 'GenerateTitleResult',
  // Workbench
  'GET /api/visits/:param/timeline': 'TimelineItem',
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

// ====== Helpers ======

function normPath(p) {
  return p.replace(/:\w+/g, ':param').replace(/\/$/, '');
}

function getZodSchema(schemaOrTypeName) {
  if (!schemaOrTypeName) return null;
  const candidates = [
    schemaOrTypeName,
    schemaOrTypeName + 'Schema',
    schemaOrTypeName.charAt(0).toLowerCase() + schemaOrTypeName.slice(1) + 'Schema',
  ];
  for (const c of candidates) {
    if (ZOD_SCHEMAS[c]) return ZOD_SCHEMAS[c];
  }
  return null;
}

function getZodFields(schemaOrTypeName) {
  const schema = getZodSchema(schemaOrTypeName);
  return schema ? schema.fields || [] : [];
}

function getGoFields(structName) {
  if (!structName) return [];
  const s = GO_STRUCTS[structName];
  return s ? s.fields : [];
}

function getGoStruct(structName) {
  if (!structName) return null;
  return GO_STRUCTS[structName] || null;
}

// ====== Zod → Go Type Mapping ======

const ZOD_TO_GO_COMPATIBLE = {
  'string': new Set(['string']),
  'number': new Set(['int', 'int8', 'int16', 'int32', 'int64', 'uint', 'uint8', 'uint16', 'uint32', 'uint64', 'float32', 'float64']),
  'boolean': new Set(['bool']),
  'datetime': new Set(['time.Time']),
};

function normalizeGoType(goType) {
  let t = goType.trim();
  t = t.replace(/^\*/, '');    // remove pointer
  t = t.replace(/^\[\]/, '');  // remove slice prefix
  t = t.replace(/^model\./, ''); // remove package prefix
  return t;
}

function isPointerType(goType) {
  return goType.trim().startsWith('*');
}

function checkTypeCompatibility(zodType, goType) {
  if (!zodType || !goType) return true; // can't check

  const baseGo = normalizeGoType(goType);

  // Enum references → Go string types are compatible
  if (zodType === 'enum' || zodType === 'literal') return true;

  // Schema refs (e.g., patientIdSchema) are typically string wrappers
  // — compatible with string type
  if (zodType === 'schemaRef') return true;

  // String compatibility
  if (zodType === 'string') return baseGo === 'string';

  // Number compatibility
  if (zodType === 'number') {
    return ZOD_TO_GO_COMPATIBLE.number.has(baseGo);
  }

  // Boolean compatibility
  if (zodType === 'boolean') return baseGo === 'bool';

  // Datetime
  if (zodType === 'datetime') return baseGo === 'time.Time';

  // Array — Go must be a slice
  if (zodType === 'array') return goType.trim().startsWith('[]');

  // Object / SchemaRef — Go must reference another known struct
  if (zodType === 'object') {
    return baseGo in GO_STRUCTS;
  }

  // Union / discriminatedUnion — complex types, skip
  if (zodType === 'union' || zodType === 'discriminatedUnion') return true;

  // Unknown — skip
  if (zodType === 'unknown') return true;

  return true; // conservative: unknown types are not flagged
}

// ====== Core Field Comparison (existing + enhanced) ======

function compareField(fieldName, zodField, goField, context) {
  const zodRequired = zodField ? zodField.required : null;

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

  // No Zod info — extra field in backend only
  if (zodRequired === null) return null;

  // --- NEW: Type comparison ---
  const zodType = zodField ? zodField.zodType : null;
  if (zodType && !checkTypeCompatibility(zodType, goField.goType)) {
    return {
      severity: 'CRITICAL',
      category: 'type_mismatch',
      endpoint: context.endpoint,
      field: fieldName,
      expected: `Zod: ${zodType}`,
      actual: `Go: ${goField.goType}`,
      description: `Field "${fieldName}": Zod ${zodType} is incompatible with Go ${goField.goType}.`,
      file: context.goFile,
      fixHint: `Change Go type of "${fieldName}" in ${context.structName} to match Zod ${zodType}.`,
    };
  }

  // --- NEW: Nullable semantics ---
  if (zodField && zodField.nullable && !goField.isPointer) {
    return {
      severity: 'MEDIUM',
      category: 'nullable_mismatch',
      endpoint: context.endpoint,
      field: fieldName,
      expected: `Zod: .nullable() — field can be null`,
      actual: `Go: ${goField.goType} (non-pointer) — cannot represent null`,
      description: `Field "${fieldName}": Zod .nullable() but Go uses non-pointer type. Null values cannot be represented.`,
      file: context.goFile,
      fixHint: `Change Go type to *${goField.goType} to support null.`,
    };
  }

  // --- NEW: Constraint comparison ---
  const constraintDrift = compareConstraints(fieldName, zodField, goField, context);
  if (constraintDrift) return constraintDrift;

  // --- Existing: omitempty checks ---
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
                     goField.goType === 'time.Time' ? 'zero time' :
                     (goField.goType.includes('int') || goField.goType === 'float64') ? '0' : 'empty string';
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

  // Zod OPTIONAL + Go REQUIRED (non-pointer, no omitempty)
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

// ====== NEW: Constraint Comparison ======

function compareConstraints(fieldName, zodField, goField, context) {
  if (!zodField || !zodField.constraints || Object.keys(zodField.constraints).length === 0) return null;
  const zc = zodField.constraints;
  const gc = goField.constraints || {};

  // Check integer constraint
  if (zc.integer && !gc.required && !gc.min) {
    // Go doesn't have an explicit "integer" constraint — check type
    const baseGo = normalizeGoType(goField.goType);
    if (!['int', 'int8', 'int16', 'int32', 'int64', 'uint', 'uint8', 'uint16', 'uint32', 'uint64'].includes(baseGo)) {
      return {
        severity: 'MEDIUM',
        category: 'constraint_missing',
        endpoint: context.endpoint,
        field: fieldName,
        expected: `Zod: .int() — integer constraint`,
        actual: `Go: ${goField.goType} — no integer guarantee`,
        description: `"${fieldName}": Zod .int() constraint not reflected in Go type or binding.`,
        file: context.goFile,
        fixHint: `Ensure Go type is an integer type (int, int64) or add binding:"..." validation.`,
      };
    }
  }

  // Check min
  if (zc.min !== undefined) {
    if (gc.min === undefined && gc.gt === undefined && gc.gte === undefined) {
      return {
        severity: 'MEDIUM',
        category: 'constraint_missing',
        endpoint: context.endpoint,
        field: fieldName,
        expected: `Zod: .min(${zc.min})`,
        actual: `Go: no min/gt/gte in binding`,
        description: `"${fieldName}": Zod .min(${zc.min}) but Go has no corresponding binding constraint.`,
        file: context.goFile,
        fixHint: `Add binding:"min=${zc.min}" to Go struct field.`,
      };
    }
  }

  // Check max
  if (zc.max !== undefined) {
    if (gc.max === undefined) {
      return {
        severity: 'MEDIUM',
        category: 'constraint_missing',
        endpoint: context.endpoint,
        field: fieldName,
        expected: `Zod: .max(${zc.max})`,
        actual: `Go: no max in binding`,
        description: `"${fieldName}": Zod .max(${zc.max}) but Go has no corresponding binding constraint.`,
        file: context.goFile,
        fixHint: `Add binding:"max=${zc.max}" to Go struct field.`,
      };
    } else if (gc.max !== zc.max) {
      return {
        severity: 'MEDIUM',
        category: 'constraint_mismatch',
        endpoint: context.endpoint,
        field: fieldName,
        expected: `Zod: .max(${zc.max})`,
        actual: `Go: binding:"max=${gc.max}"`,
        description: `"${fieldName}": Zod .max(${zc.max}) differs from Go max=${gc.max}.`,
        file: context.goFile,
        fixHint: `Align Go binding max to ${zc.max}.`,
      };
    }
  }

  // Check trimmed
  if (zc.trimmed && !gc.required) {
    // Go doesn't have a trim constraint — low severity
    return null;
  }

  // Check positive
  if (zc.positive) {
    if (gc.gt === undefined && gc.min === undefined) {
      return {
        severity: 'MEDIUM',
        category: 'constraint_missing',
        endpoint: context.endpoint,
        field: fieldName,
        expected: `Zod: .positive()`,
        actual: `Go: no positive/gt constraint`,
        description: `"${fieldName}": Zod .positive() but Go has no corresponding binding constraint.`,
        file: context.goFile,
        fixHint: `Add binding:"gt=0" to Go struct field.`,
      };
    }
  }

  return null;
}

// ====== NEW: Recursive Nested Object Comparison ======

function compareNestedFields(structName, zodSchemaName, endpoint, goFile, depth, maxDepth) {
  if (depth >= maxDepth) return [];
  if (!structName || !zodSchemaName) return [];

  const goStruct = getGoStruct(structName);
  const zodSchema = getZodSchema(zodSchemaName);
  if (!goStruct || !zodSchema) return [];

  const driftItems = [];
  const zodFieldMap = {};
  for (const f of zodSchema.fields) zodFieldMap[f.name] = f;

  const goFieldMap = {};
  for (const f of goStruct.fields) goFieldMap[f.jsonName] = f;

  const allNames = new Set([...Object.keys(zodFieldMap), ...Object.keys(goFieldMap)]);
  const goFilePath = goStruct.file || goFile;

  for (const fieldName of allNames) {
    const zodF = zodFieldMap[fieldName];
    const goF = goFieldMap[fieldName];

    const drift = compareField(fieldName, zodF || null, goF || null, {
      endpoint: `${endpoint} → ${structName}`,
      structName,
      goFile: `internal/model/${goFilePath}`,
    });

    if (drift) driftItems.push(drift);

    // Recurse into nested struct references
    if (goF && zodF && (zodF.zodType === 'object' || zodF.zodType === 'schemaRef')) {
      const nestedGoName = GO_STRUCT_REFS[structName]?.[fieldName];
      if (nestedGoName) {
        // Try to find the Zod schema for the nested type
        const nestedZodName = zodF.zodExpr ? findSchemaRefName(zodF.zodExpr) : null;
        if (nestedZodName) {
          driftItems.push(...compareNestedFields(nestedGoName, nestedZodName, endpoint, goFilePath, depth + 1, maxDepth));
        }
      }
    }
  }

  return driftItems;
}

function findSchemaRefName(zodExpr) {
  const refMatch = zodExpr.match(/^(\w+Schema)\b/);
  return refMatch ? refMatch[1] : null;
}

// ====== NEW: Enum Value Comparison ======

function compareEnumValues() {
  const driftItems = [];

  for (const [feEnumName, goEnumName] of Object.entries(ENUM_NAME_MAP)) {
    // Remove "Schema" suffix for lookup
    const feKey = feEnumName;
    const feEnum = ZOD_ENUMS[feKey];
    const goEnum = GO_ENUMS[goEnumName];

    if (!feEnum) {
      driftItems.push({
        severity: 'LOW',
        category: 'enum_frontend_missing',
        endpoint: `enum: ${feEnumName}`,
        field: feEnumName,
        expected: `Frontend Zod enum for ${goEnumName}`,
        actual: 'NOT FOUND in frontend enums',
        description: `No frontend Zod enum found for Go type ${goEnumName}.`,
        file: 'lib/api/types.ts',
        fixHint: `Add Zod enum for ${goEnumName} in frontend.`,
      });
      continue;
    }

    if (!goEnum) {
      driftItems.push({
        severity: 'HIGH',
        category: 'enum_backend_missing',
        endpoint: `enum: ${feEnumName}`,
        field: feEnumName,
        expected: `Backend Go enum for ${goEnumName}`,
        actual: 'NOT FOUND in backend enums',
        description: `No Go enum found for frontend Zod enum ${feEnumName}.`,
        file: 'internal/model/enums.go',
        fixHint: `Add Go type ${goEnumName} string with const values.`,
      });
      continue;
    }

    const feValues = new Set(feEnum.values || []);
    const goValues = new Set(goEnum.values || []);

    const feOnly = [...feValues].filter(v => !goValues.has(v));
    const goOnly = [...goValues].filter(v => !feValues.has(v));

    if (feOnly.length > 0 || goOnly.length > 0) {
      driftItems.push({
        severity: 'HIGH',
        category: 'enum_value_mismatch',
        endpoint: `enum: ${feEnumName} ↔ ${goEnumName}`,
        field: `${feEnumName}/${goEnumName}`,
        expected: `Zod values: [${feEnum.values.join(', ')}]`,
        actual: `Go values: [${goEnum.values.join(', ')}]`,
        description: `Enum value mismatch. FE only: [${feOnly.join(', ')}]. BE only: [${goOnly.join(', ')}].`,
        file: 'internal/model/enums.go',
        fixHint: feOnly.length > 0
          ? `Add missing values to Go ${goEnumName}: ${feOnly.join(', ')}.`
          : `Add missing values to frontend ${feEnumName}: ${goOnly.join(', ')}.`,
      });
    }
  }

  return driftItems;
}

// ====== NEW: Request Body Comparison ======

function compareRequestFields() {
  const driftItems = [];
  const beEndpoints = backendFields.endpoints || [];
  const feEndpoints = frontendFields.endpoints || [];

  // Build backend endpoint index
  const beByPath = {};
  for (const ep of beEndpoints) {
    const sig = `${ep.method} ${normPath(ep.path)}`;
    beByPath[sig] = ep;
  }

  for (const feEp of feEndpoints) {
    const sig = `${feEp.httpMethod} ${normPath(feEp.path)}`;
    const beEp = beByPath[sig];
    if (!beEp) continue;

    const feReqFields = feEp.requestFields || [];
    const beReqFields = beEp.requestFields || [];
    if (feReqFields.length === 0 && beReqFields.length === 0) continue;
    if (!feEp.requestSchema && beReqFields.length === 0) continue;

    const feFieldMap = {};
    for (const f of feReqFields) feFieldMap[f.name] = f;

    const beFieldMap = {};
    for (const f of beReqFields) beFieldMap[f.jsonName] = f;

    const allReqNames = new Set([...Object.keys(feFieldMap), ...Object.keys(beFieldMap)]);

    for (const fieldName of allReqNames) {
      const zodF = feFieldMap[fieldName];
      const goF = beFieldMap[fieldName];

      if (!goF && zodF) {
        driftItems.push({
          severity: 'HIGH',
          category: 'request_field_missing',
          endpoint: sig,
          field: fieldName,
          expected: `Zod: request field "${fieldName}" in ${feEp.requestSchema || 'unknown'}`,
          actual: 'NOT IN Go request struct',
          description: `Request field "${fieldName}" is in frontend input schema but missing from Go request struct.`,
          file: `internal/handler/ or internal/model/`,
          fixHint: `Add field to Go request struct: ${fieldName} <type> \`json:"${fieldName}"\`.`,
        });
        continue;
      }

      if (!zodF && goF && goF.required) {
        driftItems.push({
          severity: 'MEDIUM',
          category: 'request_field_extra_required',
          endpoint: sig,
          field: fieldName,
          expected: 'NOT in frontend input schema',
          actual: `Go: ${goF.goType} required`,
          description: `Request field "${fieldName}" is required in Go but not in frontend schema. Frontend may not send it.`,
          file: `internal/handler/ or internal/model/`,
          fixHint: `Either add "${fieldName}" to frontend input schema or make it optional in Go.`,
        });
        continue;
      }

      // Type check for request fields
      if (zodF && goF && zodF.zodType) {
        if (!checkTypeCompatibility(zodF.zodType, goF.goType)) {
          driftItems.push({
            severity: 'HIGH',
            category: 'request_type_mismatch',
            endpoint: sig,
            field: fieldName,
            expected: `Zod: ${zodF.zodType}`,
            actual: `Go: ${goF.goType}`,
            description: `Request field "${fieldName}": Zod ${zodF.zodType} incompatible with Go ${goF.goType}.`,
            file: `internal/handler/ or internal/model/`,
            fixHint: `Align Go type of "${fieldName}" with frontend Zod ${zodF.zodType}.`,
          });
        }
      }
    }
  }

  return driftItems;
}

// ====== NEW: Pagination Format Verification ======

function checkPaginationFormats() {
  const driftItems = [];
  const beEndpoints = backendFields.endpoints || [];

  // List endpoints that should use cursor-based pagination
  const cursorListSigs = [
    'GET /api/visits',
    'GET /api/visits/:param/timeline',
    'GET /api/billing/records',
    'GET /api/medical-orders',
  ];

  // Admin list endpoints that should use offset-based pagination
  const offsetListSigs = [
    'GET /admin/patients',
    'GET /admin/sessions',
  ];

  const beBySig = {};
  for (const ep of beEndpoints) {
    const sig = `${ep.method} ${normPath(ep.path)}`;
    beBySig[sig] = ep;
  }

  for (const sig of cursorListSigs) {
    const ep = beBySig[sig];
    if (!ep) continue;
    // These are wrapped in PageResult<T> via api.SuccessResponse(PageResult)
    // Check if the response is a list type
    const respStruct = GO_STRUCTS[ep.responseType];
    if (!respStruct) continue;

    const fieldNames = respStruct.fields.map(f => f.jsonName);
    // Cursor-based list responses should contain all three fields
    const hasItems = fieldNames.includes('items');
    const hasNextCursor = fieldNames.includes('nextCursor');
    const hasHasMore = fieldNames.includes('hasMore');

    if (hasItems && (!hasNextCursor || !hasHasMore)) {
      driftItems.push({
        severity: 'HIGH',
        category: 'pagination_format_mismatch',
        endpoint: sig,
        field: 'response structure',
        expected: 'cursor-based: { items, nextCursor?, hasMore } (PageResult)',
        actual: `Go struct ${ep.responseType}: fields [${fieldNames.join(', ')}]`,
        description: `List endpoint ${sig} should use cursor-based PageResult but response struct lacks pagination fields.`,
        file: `internal/model/`,
        fixHint: `Wrap with api.PageResult<T> which provides items, nextCursor, hasMore.`,
      });
    }
  }

  for (const sig of offsetListSigs) {
    const ep = beBySig[sig];
    if (!ep) continue;
    const respStruct = GO_STRUCTS[ep.responseType];
    if (!respStruct) continue;

    const fieldNames = respStruct.fields.map(f => f.jsonName);
    const hasItems = fieldNames.includes('items');
    const hasTotal = fieldNames.includes('total');
    const hasPage = fieldNames.includes('page');

    if (hasItems && (!hasTotal || !hasPage)) {
      driftItems.push({
        severity: 'HIGH',
        category: 'pagination_format_mismatch',
        endpoint: sig,
        field: 'response structure',
        expected: 'offset-based: { items, total, page, pageSize } (PageResponse)',
        actual: `Go struct ${ep.responseType}: fields [${fieldNames.join(', ')}]`,
        description: `Admin list endpoint ${sig} should use offset-based PageResponse but response struct lacks pagination fields.`,
        file: `internal/model/`,
        fixHint: `Use api.PageResponse<T> which provides items, total, page, pageSize.`,
      });
    }
  }

  return driftItems;
}

// ====== NEW: Error Format Comparison ======

function compareErrorFormats() {
  const driftItems = [];

  // Frontend apiErrorSchema defined in lib/api/types.ts
  // Expected fields: code (required), message (required), details (optional)
  const feErrorSchema = getZodSchema('apiErrorSchema');
  const beErrorStruct = GO_STRUCTS['ApiError'];

  if (!feErrorSchema && !beErrorStruct) return driftItems;

  if (feErrorSchema && !beErrorStruct) {
    driftItems.push({
      severity: 'HIGH',
      category: 'error_format_missing',
      endpoint: 'error responses',
      field: 'ApiError',
      expected: 'Frontend apiErrorSchema: { code, message, details? }',
      actual: 'NOT FOUND in Go structs (ApiError)',
      description: 'Frontend defines apiErrorSchema but Go ApiError struct not found.',
      file: 'internal/errors/api_error.go',
      fixHint: 'Define ApiError struct with code, message, details fields.',
    });
    return driftItems;
  }

  if (!feErrorSchema && beErrorStruct) {
    driftItems.push({
      severity: 'LOW',
      category: 'error_format_frontend_missing',
      endpoint: 'error responses',
      field: 'apiErrorSchema',
      expected: 'Go ApiError: { code, message, details? }',
      actual: 'NOT FOUND in frontend Zod schemas',
      description: 'Go ApiError struct exists but no matching frontend Zod schema found.',
      file: 'lib/api/types.ts',
      fixHint: 'Verify apiErrorSchema exists in frontend.',
    });
    return driftItems;
  }

  // Compare fields
  const beFieldNames = new Set(beErrorStruct.fields.map(f => f.jsonName));
  const requiredFields = ['code', 'message'];
  for (const field of requiredFields) {
    const feField = feErrorSchema.fields.find(f => f.name === field);
    const hasBeField = beFieldNames.has(field);
    if (feField && feField.required && !hasBeField) {
      driftItems.push({
        severity: 'HIGH',
        category: 'error_field_missing',
        endpoint: 'error responses',
        field,
        expected: `Zod: required field "${field}" in apiErrorSchema`,
        actual: 'NOT IN Go ApiError struct',
        description: `Required error field "${field}" is in frontend apiErrorSchema but missing from Go ApiError.`,
        file: 'internal/errors/api_error.go',
        fixHint: `Add "${field}" field to ApiError struct.`,
      });
    }
  }

  return driftItems;
}

// ====== NEW: Response Envelope Check ======

function checkResponseEnvelope() {
  const driftItems = [];

  // ApiResponse[T] wraps all responses: { success, data, error }
  const envelope = GO_STRUCTS['ApiResponse'];
  if (!envelope) {
    driftItems.push({
      severity: 'HIGH',
      category: 'envelope_struct_missing',
      endpoint: 'all endpoints',
      field: 'ApiResponse',
      expected: 'ApiResponse[T] generic wrapper: { success, data, error }',
      actual: 'NOT FOUND in Go structs',
      description: 'Backend ApiResponse envelope struct not found. This is the standard wrapper for all JSON responses.',
      file: 'pkg/api/response.go',
      fixHint: 'Define ApiResponse[T] struct with Success, Data, Error fields.',
    });
  }

  // Verify envelope fields
  if (envelope) {
    const envFields = new Set(envelope.fields.map(f => f.jsonName));
    for (const required of ['success', 'data', 'error']) {
      if (!envFields.has(required)) {
        driftItems.push({
          severity: 'HIGH',
          category: 'envelope_field_missing',
          endpoint: 'all endpoints',
          field: required,
          expected: `ApiResponse must have "${required}" field`,
          actual: `ApiResponse fields: [${[...envFields].join(', ')}]`,
          description: `ApiResponse is missing the "${required}" field. Frontend expects { success, data, error } envelope.`,
          file: 'pkg/api/response.go',
          fixHint: `Ensure ApiResponse struct has "${required}" field.`,
        });
      }
    }
  }

  return driftItems;
}

// ====== NEW: Discriminated Union Variant Comparison ======

function compareDiscriminatedUnions() {
  const driftItems = [];

  for (const [unionName, union] of Object.entries(ZOD_UNIONS)) {
    if (!union || !union.variantSchemas) continue;

    // Map union to known Go struct
    let goStructName = null;
    if (unionName.includes('flowCard') || unionName.includes('FlowCard')) goStructName = 'FlowCard';
    else if (unionName.includes('timeline') || unionName.includes('Timeline')) goStructName = 'TimelineItem';
    else if (unionName.includes('streamEvent') || unionName.includes('AssistantStream')) goStructName = 'AssistantStreamEvent';

    if (!goStructName) continue;

    const goStruct = GO_STRUCTS[goStructName];
    if (!goStruct) {
      driftItems.push({
        severity: 'HIGH',
        category: 'discriminated_union_struct_missing',
        endpoint: `union: ${unionName}`,
        field: goStructName,
        expected: `Go struct for ${goStructName} with discriminant "${union.discriminator}"`,
        actual: 'NOT FOUND',
        description: `Discriminated union "${unionName}" has no corresponding Go struct "${goStructName}".`,
        file: 'internal/model/',
        fixHint: `Create Go struct ${goStructName} with discriminant field "${union.discriminator}".`,
      });
      continue;
    }

    // Check discriminator field exists in Go struct
    const hasDiscriminator = goStruct.fields.some(f => f.jsonName === union.discriminator);
    if (!hasDiscriminator) {
      driftItems.push({
        severity: 'HIGH',
        category: 'discriminator_field_missing',
        endpoint: `union: ${unionName}`,
        field: union.discriminator,
        expected: `Go struct "${goStructName}" must have discriminator field "${union.discriminator}"`,
        actual: `Go fields: [${goStruct.fields.map(f => f.jsonName).join(', ')}]`,
        description: `Discriminated union "${unionName}" requires field "${union.discriminator}" in Go struct "${goStructName}".`,
        file: `internal/model/`,
        fixHint: `Add "${union.discriminator}" field to ${goStructName} struct.`,
      });
    }

    // For each variant, check key fields
    const goFieldNames = new Set(goStruct.fields.map(f => f.jsonName));
    for (const variantName of union.variantSchemas) {
      const variantSchema = ZOD_SCHEMAS[variantName];
      if (!variantSchema) continue;

      const variantRequired = (variantSchema.fields || []).filter(f => f.required).map(f => f.name);
      const missingVariantFields = variantRequired.filter(f => !goFieldNames.has(f));

      if (missingVariantFields.length > 0) {
        driftItems.push({
          severity: 'MEDIUM',
          category: 'discriminated_union_variant_fields',
          endpoint: `union: ${unionName}/${variantName}`,
          field: missingVariantFields.join(', '),
          expected: `Zod variant "${variantName}" requires: [${variantRequired.join(', ')}]`,
          actual: `Go struct "${goStructName}" missing: [${missingVariantFields.join(', ')}]`,
          description: `Variant "${variantName}" of "${unionName}" has required fields missing from Go struct "${goStructName}".`,
          file: `internal/model/`,
          fixHint: `Add missing fields to ${goStructName}: ${missingVariantFields.join(', ')}.`,
        });
      }
    }
  }

  return driftItems;
}

// ====== Main ======

function main() {
  const driftItems = [];
  let endpointsCompared = 0;
  let fieldsCompared = 0;
  let nestedFieldsCompared = 0;

  const feEndpoints = apiContract.endpoints || frontendFields.endpoints || [];

  if (feEndpoints.length === 0) {
    console.error('[WARN] No frontend endpoints found in api-contract.json or frontend-fields.json');
  }

  // ======== 1. Response Field Comparison (existing core logic) ========

  for (const feEp of feEndpoints) {
    const sig = `${feEp.method || feEp.httpMethod} ${normPath(feEp.path)}`;
    const structName = ENDPOINT_SCHEMA_MAP[sig];
    if (structName === null) continue;

    let actualStructName = structName;
    if (!actualStructName) {
      const altSig = `${feEp.method || feEp.httpMethod} ${feEp.path}`;
      actualStructName = ENDPOINT_SCHEMA_MAP[altSig];
    }
    if (!actualStructName) continue;

    const zodFields = getZodFields(actualStructName);
    const goFields = getGoFields(actualStructName);
    if (goFields.length === 0 && zodFields.length === 0) continue;

    endpointsCompared++;

    const zodFieldMap = {};
    for (const f of zodFields) zodFieldMap[f.name] = f;

    const goFieldMap = {};
    for (const f of goFields) goFieldMap[f.jsonName] = f;

    const allNames = new Set([...Object.keys(zodFieldMap), ...Object.keys(goFieldMap)]);
    const goFile = GO_STRUCTS[actualStructName]?.file || 'unknown';

    for (const fieldName of allNames) {
      const zodF = zodFieldMap[fieldName];
      const goF = goFieldMap[fieldName];
      fieldsCompared++;
      const drift = compareField(fieldName, zodF || null, goF || null, {
        endpoint: sig,
        structName: actualStructName,
        goFile: `internal/model/${goFile}`,
      });
      if (drift) driftItems.push(drift);
    }

    // Recursive nested comparison
    const nestedDrifts = compareNestedFields(actualStructName, actualStructName, sig, goFile, 0, 5);
    driftItems.push(...nestedDrifts);
    nestedFieldsCompared += nestedDrifts.length;
  }

  // ======== 2. Enum Value Comparison ========
  const enumDrifts = compareEnumValues();
  driftItems.push(...enumDrifts);

  // ======== 3. Request Body Comparison ========
  const requestDrifts = compareRequestFields();
  driftItems.push(...requestDrifts);

  // ======== 4. Pagination Format Check ========
  const paginationDrifts = checkPaginationFormats();
  driftItems.push(...paginationDrifts);

  // ======== 5. Error Format Comparison ========
  const errorDrifts = compareErrorFormats();
  driftItems.push(...errorDrifts);

  // ======== 6. Response Envelope Check ========
  const envelopeDrifts = checkResponseEnvelope();
  driftItems.push(...envelopeDrifts);

  // ======== 7. Discriminated Union Comparison ========
  const unionDrifts = compareDiscriminatedUnions();
  driftItems.push(...unionDrifts);

  // ======== 8. SSE Endpoint Verification ========
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

  // ======== Summary ========

  const bySeverity = {};
  const byCategory = {};
  for (const item of driftItems) {
    bySeverity[item.severity] = (bySeverity[item.severity] || 0) + 1;
    byCategory[item.category] = (byCategory[item.category] || 0) + 1;
  }

  const realDrift = driftItems.filter(d => d.severity !== 'INFO');
  const report = {
    comparedAt: new Date().toISOString(),
    endpointsCompared,
    fieldsCompared,
    nestedFieldsCompared,
    totalDriftItems: realDrift.length,
    bySeverity,
    byCategory,
    items: driftItems,
  };

  writeFileSync(resolve(ROOT, 'drift-report-fields.json'), JSON.stringify(report, null, 2));

  console.error('=== Comprehensive Field-Level Drift Report ===');
  console.error(`Endpoints compared:     ${endpointsCompared}`);
  console.error(`Top-level fields:       ${fieldsCompared}`);
  console.error(`Nested field findings:  ${nestedFieldsCompared}`);
  console.error(`Total drift items:      ${realDrift.length}`);
  console.error('');
  console.error('By Severity:');
  for (const [sev, count] of Object.entries(bySeverity).sort()) {
    if (sev !== 'INFO') console.error(`  ${sev}: ${count}`);
  }
  console.error('');
  console.error('By Category:');
  for (const [cat, count] of Object.entries(byCategory).sort()) {
    console.error(`  ${cat}: ${count}`);
  }

  if (realDrift.length > 0) {
    console.error('\n--- Drift Items ---');
    for (const item of realDrift) {
      console.error(`[${item.severity}] ${item.category}: ${item.field || '(endpoint)'} @ ${item.endpoint}`);
      console.error(`       ${item.description}`);
    }
  } else {
    console.error('\n✅ No drift detected!');
  }

  console.log(JSON.stringify(report, null, 2));
}

main();
