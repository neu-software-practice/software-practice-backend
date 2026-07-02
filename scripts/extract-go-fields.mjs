#!/usr/bin/env node
/**
 * Extract field-level Go struct information from backend source.
 * Outputs: backend-fields.json
 *
 * Key data per struct: fields with isPointer, hasOmitempty, jsonName.
 * Maps endpoints to their response AND request structs.
 * Extracts enum values, query parameters, binding constraints, and nested type refs.
 */
import { readFileSync, writeFileSync, readdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');
const INTERNAL = resolve(ROOT, 'internal');

// ====== Go Struct Parser ======

function parseGoStructs(source, filePath) {
  const structs = {};

  // Match type declarations (with optional generics), then use brace matching for the body
  const typePattern = /type\s+(\w+)\s*(?:\[[^\]]*\])?\s*struct\s*\{/g;
  let match;
  while ((match = typePattern.exec(source)) !== null) {
    const name = match[1];
    const bodyStart = match.index + match[0].length;

    // Use brace matching to find the closing }
    let depth = 1;
    let i = bodyStart;
    while (i < source.length && depth > 0) {
      if (source[i] === '{') depth++;
      else if (source[i] === '}') depth--;
      i++;
    }
    const body = source.slice(bodyStart, i - 1); // exclude closing }

    const fields = parseStructFields(body);
    structs[name] = { name, fields, file: filePath };
  }

  return structs;
}

function parseStructFields(body) {
  const fields = [];
  const lines = body.split('\n');

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    // Match: FieldName Type `json:"jsonName,omitempty" binding:"required,min=1,max=2000"`
    const taggedMatch = trimmed.match(/^(\w+)\s+([^`]+?)\s+`(?:[^`]*json:"([^"]*)"[^`]*)?(?:[^`]*binding:"([^"]*)"[^`]*)?`/);
    if (taggedMatch) {
      const goName = taggedMatch[1];
      if (goName[0] !== goName[0].toUpperCase()) continue; // unexported

      const goType = taggedMatch[2]?.trim() || 'unknown';
      const jsonTag = taggedMatch[3] || '';
      const bindingTag = taggedMatch[4] || '';

      const parts = jsonTag.split(',');
      const jsonName = parts[0] || goName;
      const hasOmitempty = parts.includes('omitempty');
      const isHidden = jsonName === '-';
      if (isHidden) continue;

      const isPointer = goType.startsWith('*');
      const isSlice = goType.startsWith('[]');

      // Parse binding constraints
      const constraints = parseBindingConstraints(bindingTag);

      fields.push({
        jsonName,
        goName,
        goType,
        isPointer,
        isSlice,
        hasOmitempty,
        required: bindingTag.includes('required'),
        binding: bindingTag,
        constraints,
      });
      continue;
    }

    // Simpler match (no tags)
    const simpleMatch = trimmed.match(/^(\w+)\s+(.+)$/);
    if (simpleMatch && simpleMatch[1][0] === simpleMatch[1][0].toUpperCase()) {
      const goName = simpleMatch[1];
      const goType = simpleMatch[2];
      fields.push({
        jsonName: goName,
        goName,
        goType,
        isPointer: goType.startsWith('*'),
        isSlice: goType.startsWith('[]'),
        hasOmitempty: false,
        required: false,
        binding: '',
        constraints: {},
      });
    }
  }

  return fields;
}

/**
 * Parse Gin binding tag constraints into structured object.
 * "required,min=1,max=2000" → { required: true, min: 1, max: 2000 }
 * "omitempty,gt=0" → { required: false, gt: 0 }
 */
function parseBindingConstraints(bindingTag) {
  const constraints = {};
  if (!bindingTag) return constraints;

  constraints.required = bindingTag.includes('required');

  const minMatch = bindingTag.match(/\bmin=(\d+)/);
  if (minMatch) constraints.min = parseInt(minMatch[1], 10);

  const maxMatch = bindingTag.match(/\bmax=(\d+)/);
  if (maxMatch) constraints.max = parseInt(maxMatch[1], 10);

  const gtMatch = bindingTag.match(/\bgt=(\d+)/);
  if (gtMatch) constraints.gt = parseInt(gtMatch[1], 10);

  const gteMatch = bindingTag.match(/\bgte=(\d+)/);
  if (gteMatch) constraints.gte = parseInt(gteMatch[1], 10);

  return constraints;
}

// ====== Go Enum Parser ======

/**
 * Parse Go typed string constants (enums).
 * Pattern:
 *   type VisitStatus string
 *   const (
 *     VisitStatusActive    VisitStatus = "active"
 *     VisitStatusCompleted VisitStatus = "completed"
 *   )
 */
function parseGoEnums(source) {
  const enums = {};

  // Match: type EnumName string (possibly preceded by comment)
  const typePattern = /type\s+(\w+)\s+string\b/g;
  let typeMatch;
  while ((typeMatch = typePattern.exec(source)) !== null) {
    const enumName = typeMatch[1];
    // Find the next const block after this type declaration
    const afterType = source.slice(typeMatch.index + typeMatch[0].length);
    const constMatch = afterType.match(/const\s*\(([\s\S]*?)\)\s*(\n|$)/);
    if (!constMatch) continue;

    const constBody = constMatch[1];
    const values = [];

    // Match: ConstName EnumName = "value"
    const valuePattern = new RegExp(`(\\w+)\\s+${enumName}\\s*=\\s*"([^"]*)"`, 'g');
    let valueMatch;
    while ((valueMatch = valuePattern.exec(constBody)) !== null) {
      values.push(valueMatch[2]);
    }

    if (values.length > 0) {
      enums[enumName] = { name: enumName, values };
    }

    // Advance past this const block to find more enums
    // (avoid re-matching the same const block for different type declarations)
  }

  return enums;
}

// ====== Router Parser: endpoint -> handler function ======

function parseRouterForEndpoints(source) {
  const endpoints = [];
  const parentMap = {};
  const prefixMap = {};

  const lines = source.split('\n');

  // Single pass: build group hierarchy
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    const gm = trimmed.match(/(\w+)\s*:=\s*(\w+)\.Group\(\s*"([^"]*)"\s*\)/);
    if (gm) {
      prefixMap[gm[1]] = gm[3]; // path prefix string (e.g., "/api")
      parentMap[gm[1]] = gm[2]; // parent variable name (e.g., "engine")
      continue;
    }
  }

  // Helper to resolve full path
  function resolvePath(groupVar, subPath) {
    let prefix = '';
    let cur = groupVar;
    const visited = new Set();
    while (cur && cur !== 'engine' && !visited.has(cur)) {
      visited.add(cur);
      prefix = (prefixMap[cur] || '') + prefix;
      cur = parentMap[cur];
    }
    return prefix + subPath;
  }

  // Second pass: extract endpoints
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    // Direct engine routes
    const em = trimmed.match(/engine\.(GET|POST|PATCH|PUT|DELETE)\(\s*"([^"]+)"/);
    if (em) {
      endpoints.push({ method: em[1], path: em[2], handler: '', isSSE: false });
      continue;
    }

    // Group routes
    const rm = trimmed.match(/(\w+)\.(GET|POST|PATCH|PUT|DELETE)\(\s*"([^"]+)"\s*,\s*router\.(\w+)\.(\w+)/);
    if (rm) {
      const path = resolvePath(rm[1], rm[3]);
      const handler = `${rm[4]}.${rm[5]}`;
      endpoints.push({ method: rm[2], path, handler, isSSE: false });
    }
  }

  return endpoints;
}

// ====== Handler Type Resolution ======

/**
 * Map handler function names to response types based on known patterns.
 */
function getResponseType(handlerName) {
  const map = {
    // Patient
    'Patient.VerifyIdentity': 'VerifyIdentityResult',
    'Patient.GetContext': 'PatientContext',
    'Patient.UpdateProfile': 'PatientProfile',
    // Visit
    'Visit.CreateSession': 'CreateSessionResult',
    'Visit.CreateFollowUp': 'CreateSessionResult',
    'Visit.ListSessions': 'VisitSessionSummary', // PageResult wrapper
    'Visit.GetSession': 'VisitSession',
    'Visit.GetSnapshot': 'VisitSnapshot',
    'Visit.SuspendVisit': 'VisitSession',
    // Auth
    'Auth.Register': 'AuthResponse',
    'Auth.Login': 'AuthResponse',
    'Auth.Refresh': 'AuthResponse',
    'Auth.Logout': 'void', // 204 no content
    // Workbench
    'Workbench.ListTimeline': 'TimelineItem', // PageResult wrapper
    'Workbench.SendMessage': 'SendMessageResult',
    'Workbench.StreamAssistantMessage': 'AssistantStreamEvent', // SSE
    'Workbench.SubmitLabDecision': 'FlowActionResult',
    'Workbench.SubmitPayment': 'FlowActionResult',
    'Workbench.SubmitFulfillment': 'FlowActionResult',
    'Workbench.SubmitTreatmentExecution': 'FlowActionResult',
    'Workbench.AckAdvice': 'FlowActionResult',
    'Workbench.AskLockedQuestion': 'AssistantStreamEvent', // SSE
    'Workbench.ClassifyIntent': 'ClassifyIntentResult',
    'Workbench.StreamConsultationReply': 'AssistantStreamEvent', // SSE
    'Workbench.ReportVitals': 'EmergencyRecheckResult',
    'Workbench.ExitVisit': 'ExitSettlementResult',
    'Workbench.ToggleTimer': 'VisitSession',
    'Workbench.DismissEmergency': 'DismissEmergencyResult',
    'Workbench.GenerateTitle': 'GenerateTitleResult',
    // Address
    'Address.ListAddresses': 'AddressListResponse',
    'Address.CreateAddress': 'Address',
    'Address.UpdateAddress': 'Address',
    'Address.DeleteAddress': 'DeleteAddressResponse',
    'Address.SetDefaultAddress': 'Address',
    // Billing
    'Billing.ListBillingRecords': 'BillingRecordsResponse',
    // Medical Order
    'MedicalOrder.ListMedicalOrders': 'MedicalOrdersResponse',
    // Admin
    'Admin.Login': 'AdminLoginResult',
    'Admin.Logout': 'AdminLogoutResult',
    'Admin.Refresh': 'AdminRefreshResult',
    'Admin.GetDashboardStats': 'DashboardStats',
    'Admin.ListPatients': 'AdminPatientListResult',
    'Admin.GetPatientDetail': 'PatientProfile',
    'Admin.ListSessions': 'AdminSessionListResult',
    'Admin.GetSessionDetail': 'VisitSession',
    'Admin.GetSettings': 'SystemSettings',
    'Admin.UpdateSettings': 'SystemSettings',
  };
  return map[handlerName] || handlerName;
}

/**
 * Map handler function names to request/input types.
 * Covers both handler-layer Request types and model-layer Input types.
 */
function getRequestType(handlerName) {
  const map = {
    // Patient
    'Patient.VerifyIdentity': 'VerifyIdentityInput',
    'Patient.UpdateProfile': 'ProfileUpdateInput',
    // Visit
    'Visit.CreateSession': 'CreateSessionInput',
    'Visit.CreateFollowUp': 'CreateFollowUpInput',
    // Auth
    'Auth.Register': 'RegisterInput',
    'Auth.Login': 'LoginInput',
    'Auth.Refresh': 'RefreshInput',
    'Auth.Logout': 'LogoutInput',
    // Workbench — handler layer (workbench_requests.go)
    'Workbench.SendMessage': 'SendMessageRequest',
    'Workbench.SubmitLabDecision': 'LabDecisionRequest',
    'Workbench.SubmitPayment': 'SubmitPaymentInput',
    'Workbench.SubmitFulfillment': 'SubmitFulfillmentInput',
    'Workbench.SubmitTreatmentExecution': 'SubmitTreatmentExecutionInput',
    'Workbench.AckAdvice': 'AckAdviceRequest',
    'Workbench.ClassifyIntent': 'ClassifyIntentRequest',
    'Workbench.ReportVitals': 'VitalsRequest',
    'Workbench.ToggleTimer': 'TimerRequest',
    'Workbench.ExitVisit': 'ExitVisitInput',
    'Workbench.DismissEmergency': 'DismissEmergencyInput',
    'Workbench.GenerateTitle': 'GenerateTitleRequest',
    'Workbench.StreamAssistantMessage': 'StreamAssistantRequest',
    'Workbench.AskLockedQuestion': 'LockQuestionRequest',
    'Workbench.StreamConsultationReply': 'ConsultRequest',
    // Address
    'Address.CreateAddress': 'CreateAddressInput',
    'Address.UpdateAddress': 'UpdateAddressInput',
    'Address.DeleteAddress': 'DeleteAddressInput',
    'Address.SetDefaultAddress': 'SetDefaultAddressInput',
    // Admin
    'Admin.Login': 'AdminLoginInput',
    'Admin.Logout': 'AdminLogoutInput',
    'Admin.Refresh': 'AdminRefreshInput',
    'Admin.UpdateSettings': 'UpdateSystemSettingsInput',
    // No request body for GET endpoints, SSE streaming handlers (body may be minimal)
  };
  return map[handlerName] || null;
}

function isSSE(handlerName) {
  return ['Workbench.StreamAssistantMessage', 'Workbench.AskLockedQuestion', 'Workbench.StreamConsultationReply'].includes(handlerName);
}

// ====== Query Parameter Extraction ======

/**
 * Extract query parameter names from handler source code.
 * Matches: c.Query("paramName"), ParseQueryInt(c, "paramName", default)
 */
function parseQueryParams(source) {
  const params = new Set();

  // c.Query("name")
  const queryRe = /c\.Query\("(\w+)"\)/g;
  let m;
  while ((m = queryRe.exec(source)) !== null) {
    params.add(m[1]);
  }

  // ParseQueryInt(c, "name", default)
  const pqIntRe = /ParseQueryInt\(c,\s*"(\w+)"/g;
  while ((m = pqIntRe.exec(source)) !== null) {
    params.add(m[1]);
  }

  // CursorFromQuery(c.Query("cursor"))
  const cursorRe = /CursorFromQuery\(c\.Query\("(\w+)"\)\)/g;
  while ((m = cursorRe.exec(source)) !== null) {
    params.add(m[1]);
  }

  return [...params].sort();
}

// ====== Struct Reference Graph ======

/**
 * Build a reference graph from struct fields to other known struct types.
 * { VisitSession: { summary: 'VisitSummary', payment: 'PaymentInfo' }, ... }
 */
function buildStructRefs(allStructs) {
  const refs = {};
  const structNames = new Set(Object.keys(allStructs));

  for (const [name, struct] of Object.entries(allStructs)) {
    const fieldRefs = {};
    for (const field of struct.fields) {
      // Clean the goType: remove pointer *, slice [], and package prefix
      let cleanType = field.goType;
      cleanType = cleanType.replace(/^\*/, '');    // remove pointer
      cleanType = cleanType.replace(/^\[\]/, '');  // remove slice
      cleanType = cleanType.replace(/^model\./, ''); // remove model. prefix
      cleanType = cleanType.trim();

      // Check if cleaned type is another known struct
      if (structNames.has(cleanType)) {
        fieldRefs[field.jsonName] = cleanType;
      }
    }
    if (Object.keys(fieldRefs).length > 0) {
      refs[name] = fieldRefs;
    }
  }

  return refs;
}

// ====== Error / Envelope / Pagination struct detection ======

/**
 * Detect which structs are error types, envelope wrappers, or pagination types.
 */
function classifyStructs(allStructs, allEnums) {
  const paginationStructs = {};
  const errorStructs = {};

  for (const [name, struct] of Object.entries(allStructs)) {
    const fieldNames = struct.fields.map(f => f.jsonName);

    // Pagination: PageResult / PageResponse patterns
    if (fieldNames.includes('items') && fieldNames.includes('hasMore')) {
      paginationStructs[name] = 'cursor';
    }
    if (fieldNames.includes('items') && fieldNames.includes('total') && fieldNames.includes('page')) {
      paginationStructs[name] = 'offset';
    }

    // Error: ApiError / SSEEventError
    if (fieldNames.includes('code') && fieldNames.includes('message') && !fieldNames.includes('accessToken')) {
      // Exclude AuthResponse which also has code-like fields
      if (!fieldNames.includes('accessToken') && !fieldNames.includes('refreshToken')) {
        errorStructs[name] = true;
      }
    }
  }

  return { paginationStructs, errorStructs };
}

// ====== Main ======

function main() {
  // 1. Parse router
  const routerSource = readFileSync(resolve(INTERNAL, 'handler/router.go'), 'utf-8');
  const routerEndpoints = parseRouterForEndpoints(routerSource);

  // 2. Parse all Go structs
  const allStructs = {};
  const dirs = [
    resolve(INTERNAL, 'model'),
    resolve(INTERNAL, 'handler'),
    resolve(INTERNAL, 'service/workbench'),
    resolve(INTERNAL, 'errors'),
    resolve(ROOT, 'pkg/api'),
  ];

  for (const dir of dirs) {
    let files;
    try { files = readdirSync(dir).filter(f => f.endsWith('.go')); } catch { continue; }
    for (const f of files) {
      const fp = resolve(dir, f);
      const source = readFileSync(fp, 'utf-8');
      Object.assign(allStructs, parseGoStructs(source, f));
    }
  }

  // 3. Parse Go enums
  let allEnums = {};
  try {
    const enumsSource = readFileSync(resolve(INTERNAL, 'model/enums.go'), 'utf-8');
    allEnums = parseGoEnums(enumsSource);
    console.error(`[ENUMS] Extracted ${Object.keys(allEnums).length} enum types`);
  } catch (err) {
    console.error(`[WARN] Could not parse enums.go: ${err.message}`);
  }

  // 4. Build struct reference graph
  const structRefs = buildStructRefs(allStructs);
  console.error(`[REFS]  Found ${Object.keys(structRefs).length} structs with nested references`);

  // 5. Classify structs (error, pagination, envelope)
  const { paginationStructs, errorStructs } = classifyStructs(allStructs, allEnums);

  // 6. Parse handler files for query params
  const handlerQueryParams = {};
  const handlerDirs = [
    resolve(INTERNAL, 'handler'),
  ];
  for (const dir of handlerDirs) {
    let files;
    try { files = readdirSync(dir).filter(f => f.endsWith('.go')); } catch { continue; }
    for (const f of files) {
      const fp = resolve(dir, f);
      const source = readFileSync(fp, 'utf-8');
      const params = parseQueryParams(source);
      if (params.length > 0) {
        handlerQueryParams[f] = params;
      }
    }
  }
  console.error(`[QUERY] Found ${Object.keys(handlerQueryParams).length} handler files with query params`);

  // 7. Enrich endpoints with response types
  const enrichedEndpoints = routerEndpoints.map(ep => {
    const responseType = getResponseType(ep.handler);
    const requestType = getRequestType(ep.handler);
    const sseFlag = isSSE(ep.handler) || ep.isSSE;

    const respStruct = allStructs[responseType];
    const respFields = respStruct ? respStruct.fields : [];
    const respFieldMap = {};
    for (const f of respFields) respFieldMap[f.jsonName] = f;

    const reqStruct = requestType ? allStructs[requestType] : null;
    const reqFields = reqStruct ? reqStruct.fields : [];
    const reqFieldMap = {};
    for (const f of reqFields) reqFieldMap[f.jsonName] = f;

    // Attach query params from handler files
    let queryParams = [];
    for (const [file, params] of Object.entries(handlerQueryParams)) {
      // Crude matching: handler files are named by domain
      queryParams = queryParams.concat(params);
    }
    // Deduplicate
    queryParams = [...new Set(queryParams)];

    // Check if this is a paginated response
    let paginationStyle = null;
    if (responseType && paginationStructs[responseType]) {
      paginationStyle = paginationStructs[responseType];
    }

    // Check if response is wrapped in ApiResponse envelope
    // All handlers use api.SuccessResponse() or api.ErrorResponse()
    const usesEnvelope = ep.method !== 'GET' || ep.path === '/api/health' ? true : null; // null = not checked statically

    return {
      ...ep,
      responseType,
      requestType,
      isSSE: sseFlag,
      responseFields: respFields,
      responseFieldMap: respFieldMap,
      structFound: !!respStruct,
      requestFields: reqFields,
      requestFieldMap: reqFieldMap,
      requestStructFound: !!reqStruct,
      queryParams,
      paginationStyle,
      usesEnvelope,
    };
  });

  const output = {
    source: 'go-backend-fields',
    extractedAt: new Date().toISOString(),
    totalEndpoints: enrichedEndpoints.length,
    endpoints: enrichedEndpoints,
    structs: allStructs,
    structRefs,
    enums: allEnums,
    paginationStructs,
    errorStructs,
    handlerQueryParams,
  };

  writeFileSync(resolve(ROOT, 'backend-fields.json'), JSON.stringify(output, null, 2));
  console.error(`[DONE] Extracted ${enrichedEndpoints.length} endpoints, ${Object.keys(allStructs).length} structs, ${Object.keys(allEnums).length} enums`);
  console.error(`[STATS] ${enrichedEndpoints.filter(e => e.structFound).length}/${enrichedEndpoints.length} endpoints have matching response structs`);
  console.error(`[STATS] ${enrichedEndpoints.filter(e => e.requestStructFound).length}/${enrichedEndpoints.length} endpoints have matching request structs`);

  // Report structs that are referenced but not found
  const missing = new Set();
  for (const ep of enrichedEndpoints) {
    if (!ep.structFound && ep.responseType !== 'void') {
      missing.add(ep.responseType);
    }
  }
  if (missing.size > 0) {
    console.error(`[WARN] Missing response structs: ${[...missing].join(', ')}`);
  }

  console.log(JSON.stringify(output, null, 2));
}

main();
