#!/usr/bin/env node
/**
 * Extract field-level Zod schema information from frontend source.
 * Outputs: frontend-fields.json
 *
 * Key data per endpoint: response field names with required/optional status.
 * Key data per schema: field list with Zod type and constraints.
 * Handles discriminatedUnion, nested objects, enums.
 */
import { readFileSync, writeFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SRC = resolve(__dirname, '..', 'src');

// ====== Schema Parser ======

/**
 * Parse a Zod object schema body to extract fields.
 * Handles: .optional(), .nullable(), .default(), z.enum(), z.array(), nested z.object()
 */
function parseZodObjectBody(source) {
  const fields = [];
  let i = 0;
  let depth = 0;

  while (i < source.length) {
    // Skip whitespace and commas
    while (i < source.length && /\s/.test(source[i])) i++;
    if (i >= source.length) break;
    if (source[i] === ',') { i++; continue; }
    if (source[i] === '}' || source[i] === ')') break;

    // Match: fieldName: zodExpression
    const fieldMatch = source.slice(i).match(/^(\w+)\s*:\s*/);
    if (!fieldMatch) { i++; continue; }

    const fieldName = fieldMatch[1];
    // Skip refines, comments
    if (fieldName === 'refine' || fieldName === 'superRefine' || fieldName === '//') {
      i += fieldMatch[0].length;
      continue;
    }

    i += fieldMatch[0].length;

    // Extract the Zod expression for this field
    const expr = extractZodExpr(source, i);
    i = expr.endIndex;

    const isOptional = expr.text.includes('.optional()');
    const isNullable = expr.text.includes('.nullable()');
    const hasDefault = expr.text.includes('.default(');
    const isRequired = !isOptional && !hasDefault;

    const zodType = classifyZodExpr(expr.text);
    const constraints = extractExprConstraints(expr.text);

    fields.push({
      name: fieldName,
      zodExpr: expr.text.substring(0, 80),
      zodType,
      required: isRequired,
      optional: isOptional,
      nullable: isNullable,
      hasDefault,
      constraints,
    });
  }

  return fields;
}

/** Extract a balanced Zod expression starting at index i */
function extractZodExpr(source, start) {
  let i = start;
  let depth = 0;
  let inString = false;
  let stringChar = '';

  while (i < source.length) {
    const ch = source[i];

    if (inString) {
      if (ch === '\\') { i += 2; continue; }
      if (ch === stringChar) { inString = false; }
      i++;
      continue;
    }

    if (ch === '"' || ch === "'" || ch === '`') {
      inString = true;
      stringChar = ch;
      i++;
      continue;
    }

    if (ch === '(' || ch === '{' || ch === '[') { depth++; }
    else if (ch === ')' || ch === '}' || ch === ']') { depth--; }

    // End of expression: comma, closing brace/paren at depth 0, or line end
    if (depth < 0) {
      break;
    }
    // Only break on comma or closing brace at depth 0 (field delimiters).
    // Don't break on ')' because it could be z.string().optional() ending.
    if (depth === 0 && (ch === ',' || ch === '}')) {
      break;
    }

    i++;
  }

  return { text: source.slice(start, i), endIndex: i };
}

/** Classify a Zod expression into a high-level type */
function classifyZodExpr(expr) {
  if (expr.includes('z.string()')) return 'string';
  if (expr.includes('z.number()') || expr.includes('z.coerce.number()')) return 'number';
  if (expr.includes('z.boolean()')) return 'boolean';
  if (expr.includes('z.array(')) return 'array';
  if (expr.includes('z.object(')) return 'object';
  if (expr.includes('z.enum(')) return 'enum';
  if (expr.includes('z.literal(')) return 'literal';
  if (expr.includes('z.discriminatedUnion(')) return 'discriminatedUnion';
  if (expr.includes('z.union(')) return 'union';
  if (expr.includes('z.unknown(')) return 'unknown';
  if (expr.includes('z.date(') || expr.includes('datetime()')) return 'datetime';
  if (expr.includes('z.instanceof(Date)')) return 'date';
  // Reference to another schema
  if (/^\w+Schema\b/.test(expr.trim())) return 'schemaRef';
  return 'unknown';
}

function extractExprConstraints(expr) {
  const constraints = [];
  if (expr.includes('.min(')) constraints.push('hasMin');
  if (expr.includes('.max(')) constraints.push('hasMax');
  if (expr.includes('.regex(')) constraints.push('hasRegex');
  if (expr.includes('.trim()')) constraints.push('trimmed');
  if (expr.includes('.positive()')) constraints.push('positive');
  if (expr.includes('.int()')) constraints.push('integer');
  return constraints;
}

// ====== Discriminated Union Parser ======

function parseDiscriminatedUnion(source, filePath) {
  const unions = {};
  const pattern = /export\s+const\s+(\w+)\s*=\s*z\.discriminatedUnion\(\s*["'`](\w+)["'`]\s*,\s*\[([\s\S]*?)\]\s*\)/g;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    const name = match[1];
    const discriminator = match[2];
    const variantsBlock = match[3];
    const variantNames = [];
    const vPattern = /(\w+Schema)/g;
    let vm;
    while ((vm = vPattern.exec(variantsBlock)) !== null) {
      variantNames.push(vm[1]);
    }
    unions[name] = { name, discriminator, variantSchemas: variantNames, file: filePath };
  }
  return unions;
}

// ====== Enum Parser ======

function parseZodEnums(source) {
  const enums = {};
  const pattern = /export\s+const\s+(\w+)\s*=\s*z\.enum\(\s*\[([^\]]+)\]\s*\)/g;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    const name = match[1];
    const values = match[2]
      .split(',')
      .map(v => v.trim().replace(/["'`]/g, ''))
      .filter(Boolean);
    enums[name] = { name, values };
  }
  return enums;
}

// ====== API Facade Parser (endpoint -> response schema) ======

function parseApiFacadeForTypes(source) {
  const mappings = [];

  // Collapse to single line for easier parsing
  const flat = source.replace(/\n\s*/g, ' ');

  // Match each method definition + its transport call as a unit
  // Pattern: methodName(...): Promise<ReturnType> { ... transport.method(`/path` ... }
  const methodBlockPattern = /(\w+)\s*\([^)]*\)\s*:\s*Promise<([^>]+(?:<[^>]+>)*[^>]*)>\s*\{[^}]*?\.(post|get|patch|put|delete|stream)\b[^(]*\(\s*["'`]([\/][^"'`]+)["'`]/g;

  let match;
  while ((match = methodBlockPattern.exec(flat)) !== null) {
    const facadeMethod = match[1];
    const returnType = match[2].trim();
    const httpVerb = match[3];
    const rawPath = match[4];

    const method = httpVerb === 'stream' ? 'STREAM' : httpVerb.toUpperCase();
    const isSSE = method === 'STREAM';

    // Normalize path
    let normalized = rawPath;
    normalized = normalized.replace(/\$\{(?:body|input|query)\.(\w+)\}/g, ':$1');
    normalized = normalized.replace(/\$\{(\w+)\}/g, ':$1');
    if (!normalized.startsWith('/admin') && !normalized.startsWith('/api')) {
      normalized = '/api' + normalized;
    }

    const schemaName = inferResponseSchema(facadeMethod, returnType);

    mappings.push({
      facadeMethod,
      httpMethod: isSSE ? 'POST' : method,
      path: normalized,
      isSSE,
      returnType,
      responseSchema: schemaName,
    });
  }

  return mappings;
}

function inferResponseSchema(methodName, returnType) {
  // Map known return types to schema names
  const schemaMap = {
    'VisitSession': 'visitSessionSchema',
    'VisitSnapshot': 'visitSnapshotSchema',
    'CreateSessionResult': 'createSessionResultSchema',
    'VisitSessionSummary': 'visitSessionSummarySchema',
    'FlowActionResult': 'flowActionResultSchema',
    'PatientProfile': 'patientProfileSchema',
    'PatientContext': 'patientContextSchema',
    'VerifyIdentityResult': 'verifyIdentityResultSchema',
    'AuthResponse': 'authResponseSchema',
    'DashboardStats': 'dashboardStatsSchema',
    'SystemSettings': 'systemSettingsSchema',
    'Address': 'addressSchema',
    'BillingRecordsResponse': 'billingRecordsResponseSchema', // might be listBillingRecordsResultSchema
    'MedicalOrdersResponse': 'medicalOrdersResponseSchema',   // might be listMedicalOrdersResultSchema
    'SendMessageResult': 'sendMessageResultSchema',
    'EmergencyRecheckResult': 'emergencyRecheckResultSchema',
    'ClassifyIntentResult': 'classifyIntentResultSchema',
    'ExitSettlementResult': 'exitSettlementResultSchema',
    'AdminLoginResult': 'adminLoginResultSchema',
    'VisitSummary': 'visitSummarySchema',
    'AdminPatientItem': 'adminPatientItemSchema',
    'AdminSessionItem': 'adminSessionItemSchema',
    'PaginatedResponse': 'paginatedResponseSchema',
  };

  // Check method name patterns
  const methodMap = {
    'getSession': 'visitSessionSchema',
    'getReadonlySnapshot': 'visitSnapshotSchema',
    'createSession': 'createSessionResultSchema',
    'createFollowUp': 'createSessionResultSchema',
    'listSessions': 'visitSessionSummarySchema', // in PageResult
    'getPatientContext': 'patientContextSchema',
    'verifyIdentity': 'verifyIdentityResultSchema',
    'updatePatientProfile': 'patientProfileSchema',
    'sendMessage': 'sendMessageResultSchema',
    'submitLabDecision': 'flowActionResultSchema',
    'submitPayment': 'flowActionResultSchema',
    'submitFulfillment': 'flowActionResultSchema',
    'submitTreatmentExecution': 'flowActionResultSchema',
    'ackAdvice': 'flowActionResultSchema',
    'reportVitals': 'emergencyRecheckResultSchema',
    'classifyFollowUpIntent': 'classifyIntentResultSchema',
    'exitVisit': 'exitSettlementResultSchema',
    'pauseVisitTimer': 'visitSessionSchema',
    'resumeVisitTimer': 'visitSessionSchema',
    'toggleTimer': 'visitSessionSchema',
    'dismissEmergency': 'dismissEmergencyResultSchema',
    'suspendVisit': 'suspendVisitResultSchema',
    'listAddresses': 'addressListResponseSchema',
    'createAddress': 'addressSchema',
    'updateAddress': 'addressSchema',
    'deleteAddress': 'deleteAddressResponseSchema',
    'setDefaultAddress': 'addressSchema',
    'listBillingRecords': 'listBillingRecordsResultSchema',
    'listMedicalOrders': 'listMedicalOrdersResultSchema',
    'generateTitle': 'generateTitleResultSchema',
    'getDashboardStats': 'dashboardStatsSchema',
    'listPatients': 'adminPatientListResultSchema',
    'getPatient': 'patientProfileSchema',
    'listSessions': 'adminSessionListResultSchema',
    'getSettings': 'systemSettingsSchema',
    'updateSettings': 'systemSettingsSchema',
    'login': 'authResponseSchema',
    'register': 'authResponseSchema',
    'refresh': 'authResponseSchema',
  };

  if (schemaMap[returnType]) return schemaMap[returnType];
  if (methodMap[methodName]) return methodMap[methodName];

  return returnType || 'unknown';
}

// ====== Main ======

function main() {
  // 1. Parse all schema files
  const allSchemas = {};
  const allUnions = {};
  const allEnums = {};

  const schemaFiles = [
    'features/workbench/api/timeline-schemas.ts',
    'features/workbench/api/schemas.ts',
    'features/patient/api/schemas.ts',
    'features/patient/api/address-schemas.ts',
    'features/visits/api/schemas.ts',
    'features/auth/api/schemas.ts',
    'features/billing/api/schemas.ts',
    'features/medical-orders/api/schemas.ts',
    'features/admin/api/schemas.ts',
    'lib/api/types.ts',
  ];

  for (const f of schemaFiles) {
    const fp = resolve(SRC, f);
    let source;
    try { source = readFileSync(fp, 'utf-8'); } catch { continue; }

    // Parse exported schemas using brace matching
    // Find all: export const xxxSchema = z.object({
    const schemaStarts = [];
    const startPattern = /export\s+const\s+(\w+)\s*=\s*z\.object\(\{/g;
    let sm;
    while ((sm = startPattern.exec(source)) !== null) {
      schemaStarts.push({ name: sm[1], start: sm.index + sm[0].length - 1 }); // position at opening {
    }

    for (const ss of schemaStarts) {
      // Find matching closing } and check for modifiers
      const bodyStart = ss.start + 1; // after {
      let depth = 1;
      let i = bodyStart;
      while (i < source.length && depth > 0) {
        if (source[i] === '{' || source[i] === '(' || source[i] === '[') depth++;
        else if (source[i] === '}' || source[i] === ')' || source[i] === ']') depth--;
        i++;
      }
      const body = source.slice(bodyStart, i - 1); // exclude closing }

      // Check for modifiers after closing }: .strict(), .partial(), .passthrough()
      let modifier = '';
      const after = source.slice(i).trimStart();
      const modMatch = after.match(/^\.(\w+)\(\)/);
      if (modMatch) modifier = modMatch[1];

      allSchemas[ss.name] = {
        name: ss.name,
        fields: parseZodObjectBody(body),
        file: f,
        kind: 'object',
        modifier,
      };
    }

    // Parse discriminatedUnions
    Object.assign(allUnions, parseDiscriminatedUnion(source, f));

    // Parse enums
    Object.assign(allEnums, parseZodEnums(source));
  }

  // 2. Parse API facades
  const apiFiles = [
    'features/workbench/api/index.ts',
    'features/patient/api/index.ts',
    'features/visits/api/index.ts',
    'features/auth/api/auth-api.ts',
    'features/billing/api/index.ts',
    'features/medical-orders/api/index.ts',
    'features/admin/api/admin-api.ts',
  ];

  const endpointMappings = [];
  for (const f of apiFiles) {
    const fp = resolve(SRC, f);
    let source;
    try { source = readFileSync(fp, 'utf-8'); } catch { continue; }
    endpointMappings.push(...parseApiFacadeForTypes(source));
  }

  // 3. Enrich endpoints with schema fields
  const enrichedEndpoints = endpointMappings.map(ep => {
    const schema = allSchemas[ep.responseSchema];
    const fields = schema ? schema.fields : [];
    const fieldMap = {};
    for (const f of fields) fieldMap[f.name] = f;

    return {
      ...ep,
      schemaName: ep.responseSchema,
      responseFields: fields,
      responseFieldMap: fieldMap,
      schemaFound: !!schema,
    };
  });

  const output = {
    source: 'frontend-zod-fields',
    extractedAt: new Date().toISOString(),
    totalEndpoints: enrichedEndpoints.length,
    endpoints: enrichedEndpoints,
    schemas: allSchemas,
    unions: allUnions,
    enums: allEnums,
  };

  writeFileSync(resolve(__dirname, '..', 'frontend-fields.json'), JSON.stringify(output, null, 2));
  console.error(`[DONE] Extracted ${enrichedEndpoints.length} endpoints, ${Object.keys(allSchemas).length} schemas, ${Object.keys(allUnions).length} unions, ${Object.keys(allEnums).length} enums`);
  console.error(`[STATS] ${enrichedEndpoints.filter(e => e.schemaFound).length}/${enrichedEndpoints.length} endpoints have matching schemas`);
  console.log(JSON.stringify(output, null, 2));
}

main();
