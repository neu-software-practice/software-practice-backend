#!/usr/bin/env node
/**
 * Extract field-level Go struct information from backend source.
 * Outputs: backend-fields.json
 *
 * Key data per struct: fields with isPointer, hasOmitempty, jsonName.
 * Maps endpoints to their response structs.
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

  // Match type declarations with their full body
  const pattern = /type\s+(\w+)\s+struct\s*\{([^}]*)\}/gs;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    const name = match[1];
    const body = match[2];
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

    // Match: FieldName Type `json:"jsonName,omitempty" binding:"required"`
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

      fields.push({
        jsonName,
        goName,
        goType,
        isPointer,
        isSlice,
        hasOmitempty,
        required: bindingTag.includes('required'),
        binding: bindingTag,
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
      });
    }
  }

  return fields;
}

// ====== Router Parser: endpoint -> handler function ======

function parseRouterForEndpoints(source) {
  const endpoints = [];
  const parentMap = {};
  const prefixMap = {};

  const lines = source.split('\n');

  // First pass: build group hierarchy
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    const gm = trimmed.match(/(\w+)\s*:=\s*(\w+)\.Group\(\s*"([^"]*)"\s*\)/);
    if (gm) {
      prefixMap[gm[1]] = gm[2];
      parentMap[gm[1]] = gm[3] || gm[2]; // parent var name
      continue;
    }
  }

  // Fix parentMap: "api := engine.Group(...)" → parent is null
  // "authGroup := api.Group(...)" → parent is "api"
  // Need to capture correct parent: gm[2] is the parent variable
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    const gm = trimmed.match(/(\w+)\s*:=\s*(\w+)\.Group\(\s*"([^"]*)"\s*\)/);
    if (gm) {
      parentMap[gm[1]] = gm[2]; // gm[2] is the parent variable name
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
 * This is a hardcoded mapping derived from analyzing the codebase.
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

function isSSE(handlerName) {
  return ['Workbench.StreamAssistantMessage', 'Workbench.AskLockedQuestion', 'Workbench.StreamConsultationReply'].includes(handlerName);
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

  // 3. Enrich endpoints with response types
  const enrichedEndpoints = routerEndpoints.map(ep => {
    const responseType = getResponseType(ep.handler);
    const sseFlag = isSSE(ep.handler) || ep.isSSE;
    const struct = allStructs[responseType];
    const fields = struct ? struct.fields : [];
    const fieldMap = {};
    for (const f of fields) fieldMap[f.jsonName] = f;

    return {
      ...ep,
      responseType,
      isSSE: sseFlag,
      responseFields: fields,
      responseFieldMap: fieldMap,
      structFound: !!struct,
    };
  });

  const output = {
    source: 'go-backend-fields',
    extractedAt: new Date().toISOString(),
    totalEndpoints: enrichedEndpoints.length,
    endpoints: enrichedEndpoints,
    structs: allStructs,
  };

  writeFileSync(resolve(ROOT, 'backend-fields.json'), JSON.stringify(output, null, 2));
  console.error(`[DONE] Extracted ${enrichedEndpoints.length} endpoints, ${Object.keys(allStructs).length} structs`);
  console.error(`[STATS] ${enrichedEndpoints.filter(e => e.structFound).length}/${enrichedEndpoints.length} endpoints have matching response structs`);

  // Report structs that are referenced but not found
  const missing = new Set();
  for (const ep of enrichedEndpoints) {
    if (!ep.structFound && ep.responseType !== 'void') {
      missing.add(ep.responseType);
    }
  }
  if (missing.size > 0) {
    console.error(`[WARN] Missing structs: ${[...missing].join(', ')}`);
  }

  console.log(JSON.stringify(output, null, 2));
}

main();
