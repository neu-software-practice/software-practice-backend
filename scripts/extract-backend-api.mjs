#!/usr/bin/env node
/**
 * Extract API Structure from Go Backend - rewritten with precise Gin route parsing.
 */
import { readFileSync, writeFileSync, readdirSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');
const INTERNAL = resolve(ROOT, 'internal');

// ---------------------------------------------------------------------------
// 1. Parse Router with group nesting tracking
// ---------------------------------------------------------------------------

function parseRouter(source) {
  const endpoints = [];
  const parentMap = {}; // varName → parentVarName
  const prefixMap = {}; // varName → prefix

  const lines = source.split('\n');

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    // Match: varName := parent.Group("prefix")
    const groupMatch = trimmed.match(/(\w+)\s*:=\s*(\w+)\.Group\(\s*"([^"]*)"\s*\)/);
    if (groupMatch) {
      const varName = groupMatch[1];
      const parentVar = groupMatch[2];
      const prefix = groupMatch[3];
      prefixMap[varName] = prefix;
      parentMap[varName] = parentVar;
      continue;
    }

    // Direct engine routes: engine.GET("/api/health", ...)
    const engineMatch = trimmed.match(/engine\.(GET|POST|PATCH|PUT|DELETE)\(\s*"([^"]+)"/);
    if (engineMatch) {
      endpoints.push({
        method: engineMatch[1],
        fullPath: engineMatch[2],
        handler: '',
        isSSE: false,
      });
      continue;
    }

    // Group routes: groupVar.METHOD("/path", router.Handler.Method)
    const routeMatch = trimmed.match(/(\w+)\.(GET|POST|PATCH|PUT|DELETE)\(\s*"([^"]+)"\s*,\s*router\.(\w+)\.(\w+)/);
    if (routeMatch) {
      const groupVar = routeMatch[1];
      const method = routeMatch[2];
      const subPath = routeMatch[3];
      const handlerGroup = routeMatch[4];
      const handlerMethod = routeMatch[5];

      // Resolve full prefix by walking up the parent chain
      let fullPrefix = '';
      let cur = groupVar;
      while (cur && cur !== 'engine') {
        fullPrefix = (prefixMap[cur] || '') + fullPrefix;
        cur = parentMap[cur];
      }

      endpoints.push({
        method,
        fullPath: fullPrefix + subPath,
        handler: `${handlerGroup}.${handlerMethod}`,
        isSSE: false,
      });
    }
  }

  return endpoints;
}

// ---------------------------------------------------------------------------
// 2. Parse Structs
// ---------------------------------------------------------------------------

function parseGoStructs(source, filePath) {
  const structs = {};

  const structPattern = /type\s+(\w+)\s+struct\s*\{([^}]+)\}/gs;
  let match;
  while ((match = structPattern.exec(source)) !== null) {
    const name = match[1];
    const body = match[2];
    const fields = parseStructFields(body);
    if (fields.length > 0) {
      structs[name] = { name, fields, file: filePath };
    }
  }
  return structs;
}

function parseStructFields(body) {
  const fields = [];
  const lines = body.split('\n');

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//')) continue;

    // FieldName Type `json:"fieldName" binding:"required,..."`
    const match = trimmed.match(/^(\w+)\s+(.+?)\s+`(?:.*?json:"([^"]*)".*?)?(?:.*?binding:"([^"]*)".*?)?`/);
    if (match) {
      const goName = match[1];
      if (goName[0] !== goName[0].toUpperCase()) continue; // unexported

      const goType = match[2].trim();
      const jsonTag = match[3] || '';
      const jsonName = jsonTag.split(',')[0] || goName;
      const bindingTag = match[4] || '';

      fields.push({
        name: jsonName,
        goName,
        goType,
        type: mapGoType(goType),
        required: bindingTag.includes('required'),
        optional: goType.startsWith('*') || jsonTag.includes('omitempty'),
        binding: bindingTag,
      });
      continue;
    }

    // Simpler match without backtick tags
    const simpleMatch = trimmed.match(/^(\w+)\s+(.+)$/);
    if (simpleMatch && simpleMatch[1][0] === simpleMatch[1][0].toUpperCase()) {
      fields.push({
        name: simpleMatch[1],
        goName: simpleMatch[1],
        goType: simpleMatch[2],
        type: mapGoType(simpleMatch[2]),
        required: false,
        optional: simpleMatch[2].startsWith('*'),
        binding: '',
      });
    }
  }

  return fields;
}

function mapGoType(goType) {
  const isPtr = goType.startsWith('*');
  const base = isPtr ? goType.slice(1) : goType;

  const m = {
    'string': 'string',
    'int': 'number',
    'int64': 'number',
    'uint': 'number',
    'float64': 'number',
    'bool': 'boolean',
    'time.Time': 'string (datetime)',
    'uuid.UUID': 'string',
    'json.RawMessage': 'object',
  };

  let result = m[base] || base;
  if (base.startsWith('[]')) result = `array<${mapGoType(base.slice(2))}>`;
  if (base.startsWith('map[')) result = 'object';
  if (isPtr) result = `${result}?`;

  return result;
}

// ---------------------------------------------------------------------------
// 3. Parse Enums from enums.go
// ---------------------------------------------------------------------------

function parseGoEnums(source) {
  const enums = {};

  // Pattern 1: typed string constants
  // type VisitStatus string
  // const (
  //   VisitStatusChatting VisitStatus = "chatting"
  //   ...
  // )
  const typePattern = /type\s+(\w+)\s+(string|int)\s*\/\/\s*(.*)/g;
  let match;

  while ((match = typePattern.exec(source)) !== null) {
    const name = match[1];
    const kind = match[2];
    const comment = match[3]?.trim() || '';

    // Find const block after this type
    const afterType = source.slice(match.index);
    const constMatch = afterType.match(/const\s*\(\s*([\s\S]*?)\)/);
    if (constMatch) {
      const block = constMatch[1];
      const values = [];
      const valPattern = /(\w+)\s+(?:\w+)\s*=\s*"([^"]+)"/g;
      let vm;
      while ((vm = valPattern.exec(block)) !== null) {
        values.push(vm[2]);
      }
      if (values.length > 0) {
        enums[name] = { name, kind, values, description: comment };
      }
    }
  }

  // Pattern 2: iota-based int enums
  // type VisitMachineState int
  // const (
  //   VisitMachineStateInit VisitMachineState = iota
  //   ...
  // )
  // We'll extract the constant names as values

  return enums;
}

// ---------------------------------------------------------------------------
// 4. Parse Error Codes
// ---------------------------------------------------------------------------

function parseErrorCodes(source) {
  const codes = [];
  const pattern = /(\w+)\s*=\s*"([^"]+)"/g;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    codes.push({ constName: match[1], code: match[2] });
  }
  return codes;
}

// ---------------------------------------------------------------------------
// 5. Main
// ---------------------------------------------------------------------------

function main() {
  // Parse router
  const routerSource = readFileSync(resolve(INTERNAL, 'handler/router.go'), 'utf-8');
  const endpoints = parseRouter(routerSource);
  console.error(`[INFO] router.go: ${endpoints.length} endpoints`);

  // Parse models
  const allStructs = {};
  const modelDir = resolve(INTERNAL, 'model');
  const files = readdir(modelDir).filter(f => f.endsWith('.go'));
  for (const f of files) {
    const source = readFileSync(resolve(modelDir, f), 'utf-8');
    Object.assign(allStructs, parseGoStructs(source, f));
  }

  // Parse handler structs
  const handlerDir = resolve(INTERNAL, 'handler');
  const handlerFiles = readdir(handlerDir).filter(f => f.endsWith('.go'));
  for (const f of handlerFiles) {
    const source = readFileSync(resolve(handlerDir, f), 'utf-8');
    Object.assign(allStructs, parseGoStructs(source, f));
  }
  console.error(`[INFO] structs: ${Object.keys(allStructs).length}`);

  // Parse enums
  const enumSource = readFileSync(resolve(modelDir, 'enums.go'), 'utf-8');
  const enums = parseGoEnums(enumSource);
  console.error(`[INFO] enums: ${Object.keys(enums).length}`);

  // Parse error codes
  const codesSource = readFileSync(resolve(INTERNAL, 'errors/codes.go'), 'utf-8');
  const errorCodes = parseErrorCodes(codesSource);
  console.error(`[INFO] error codes: ${errorCodes.length}`);

  const result = {
    source: 'go-backend',
    extractedAt: new Date().toISOString(),
    totalEndpoints: endpoints.length,
    endpoints,
    structs: allStructs,
    enums,
    errorCodes,
  };

  writeFileSync(resolve(ROOT, 'backend-api.json'), JSON.stringify(result, null, 2));
  console.error(`\n[DONE] backend-api.json written`);
  console.log(JSON.stringify(result, null, 2));
}

function readdir(dir) {
  try { return readdirSync(dir); } catch { return []; }
}

main();
