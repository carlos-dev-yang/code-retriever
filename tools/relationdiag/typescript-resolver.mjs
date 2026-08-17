#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { createRequire } from "node:module";

const protocol = "cidx.relation-diagnostic.v1";
const version = "6.0.3";

function fail(message) { process.stderr.write(`${message}\n`); process.exitCode = 1; }
function relative(value) { return typeof value === "string" && value !== "" && !path.isAbsolute(value) && !value.includes("\\") && path.posix.normalize(value) === value && value !== ".." && !value.startsWith("../"); }
function result(id, outcome, target = null) {
  const value = { protocol, id, outcome, typescript_version: version, resolver_scope: "indexed-universe-v1" };
  if (target) Object.assign(value, target);
  process.stdout.write(`${JSON.stringify(value)}\n`);
}
function byteOffset(text, byte) {
  if (!Number.isInteger(byte) || byte < 0) return -1;
  let bytes = 0;
  for (let offset = 0; offset <= text.length;) {
    if (bytes === byte) return offset;
    if (offset === text.length) return -1;
    const code = text.codePointAt(offset);
    const width = code > 0xffff ? 2 : 1;
    bytes += Buffer.byteLength(text.slice(offset, offset + width), "utf8");
    offset += width;
  }
  return -1;
}
function byteRange(text, node, ts) {
  return [Buffer.byteLength(text.slice(0, node.getStart()), "utf8"), Buffer.byteLength(text.slice(0, node.getEnd()), "utf8")];
}
function nodeAtRange(sourceFile, candidate, ts) {
  let found = null;
  const visit = node => {
    const [start, end] = byteRange(sourceFile.text, node, ts);
    if (start === candidate.start_byte && end === candidate.end_byte) {
      if (candidate.kind === "CALLS" && ts.isCallExpression(node)) found = node;
      if (candidate.kind === "TYPE_REF" && ts.isIdentifier(node)) found = node;
		if (candidate.kind === "MEMBER_OF" && (ts.isMethodDeclaration(node) || ts.isPropertyDeclaration(node) || ts.isMethodSignature(node) || ts.isPropertySignature(node))) found = node;
    }
    if (!found) ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return found;
}
function symbolFor(node, candidate, checker, ts) {
  if (candidate.kind === "CALLS") return checker.getSymbolAtLocation(node.expression);
	if (candidate.kind === "MEMBER_OF") {
		let container = node.parent;
		while (container && !ts.isClassDeclaration(container) && !ts.isInterfaceDeclaration(container)) container = container.parent;
		return container && container.name ? checker.getSymbolAtLocation(container.name) : undefined;
	}
  return checker.getSymbolAtLocation(node);
}
function unalias(symbol, checker, ts) { return symbol && (symbol.flags & ts.SymbolFlags.Alias) ? checker.getAliasedSymbol(symbol) : symbol; }
function main(input) {
  if (!input || input.protocol !== protocol || !path.isAbsolute(input.typescript_root) || !path.isAbsolute(input.source_root) || !Array.isArray(input.files) || !Array.isArray(input.candidates)) throw new Error("invalid resolver request");
  const require = createRequire(path.join(input.typescript_root, "package.json"));
  const ts = require("typescript");
  if (ts.version !== version) throw new Error(`typescript version drift: ${ts.version}`);
  const universe = new Map();
  const canonicalRoot = fs.realpathSync(input.source_root);
  for (const file of input.files) {
    if (!relative(file.path) || !["typescript", "tsx"].includes(file.language) || !/^[a-f0-9]{64}$/.test(file.indexed_sha256 || "") || universe.has(file.path)) throw new Error("invalid indexed TypeScript universe");
    const absolute = path.resolve(input.source_root, file.path);
    const canonical = fs.realpathSync(absolute);
    if (canonical !== canonicalRoot && !canonical.startsWith(`${canonicalRoot}${path.sep}`)) throw new Error("indexed file escapes source root");
    universe.set(file.path, canonical);
  }
  for (const [relativePath, absolutePath] of universe) {
    const digest = crypto.createHash("sha256").update(fs.readFileSync(absolutePath)).digest("hex");
    const expected = input.files.find(file => file.path === relativePath).indexed_sha256;
    if (digest !== expected) throw new Error(`indexed source hash mismatch: ${relativePath}`);
  }
  const rootNames = [...universe.values()];
  // The controller pins this exact root config.  Do not search parents, which
  // could silently inherit a project configuration outside the indexed corpus.
  const configPath = path.join(canonicalRoot, "tsconfig.json");
  const config = fs.existsSync(configPath) ? ts.readConfigFile(configPath, ts.sys.readFile) : { config: {} };
  if (config.error) throw new Error("invalid tsconfig");
  if (config.config && typeof config.config.extends === "string") {
    const extended = path.resolve(path.dirname(configPath), config.config.extends);
    const canonicalExtended = fs.realpathSync(extended);
    if (canonicalExtended !== canonicalRoot && !canonicalExtended.startsWith(`${canonicalRoot}${path.sep}`)) throw new Error("tsconfig extends outside indexed source root");
  }
  const parsed = ts.parseJsonConfigFileContent(config.config || {}, ts.sys, input.source_root, { noEmit: true }, configPath);
  const options = { ...parsed.options, noEmit: true, allowJs: false };
  const defaultHost = ts.createCompilerHost(options);
  const compilerLibRoot = fs.realpathSync(path.dirname(ts.getDefaultLibFilePath(options)));
  const allowedFile = fileName => {
    try {
      const canonical = fs.realpathSync(fileName);
      if ([...universe.values()].includes(canonical)) return true;
      return canonical.startsWith(`${compilerLibRoot}${path.sep}`) && canonical.endsWith(".d.ts");
    } catch { return false; }
  };
  const host = { ...defaultHost };
  const sourceFile = defaultHost.getSourceFile.bind(defaultHost);
  const readFile = defaultHost.readFile.bind(defaultHost);
  const fileExists = defaultHost.fileExists.bind(defaultHost);
  host.fileExists = name => allowedFile(name) && fileExists(name);
  host.readFile = name => allowedFile(name) ? readFile(name) : undefined;
  host.getSourceFile = (name, languageVersion, onError, shouldCreateNewSourceFile) => allowedFile(name) ? sourceFile(name, languageVersion, onError, shouldCreateNewSourceFile) : undefined;
  const program = ts.createProgram({ rootNames, options, host });
  for (const source of program.getSourceFiles()) {
    const canonical = fs.realpathSync(source.fileName);
    if (![...universe.values()].includes(canonical) && !(canonical.startsWith(`${compilerLibRoot}${path.sep}`) && canonical.endsWith(".d.ts"))) throw new Error("Program loaded source outside indexed universe");
  }
  const checker = program.getTypeChecker();
  const ids = new Set();
  for (const candidate of input.candidates) {
    if (!relative(candidate.id) || !relative(candidate.path) || !["typescript", "tsx"].includes(candidate.language) || !["CALLS", "TYPE_REF", "MEMBER_OF"].includes(candidate.kind) || !Number.isInteger(candidate.start_byte) || !Number.isInteger(candidate.end_byte) || candidate.end_byte <= candidate.start_byte || ids.has(candidate.id)) throw new Error("invalid candidate");
    ids.add(candidate.id);
    const absolute = universe.get(candidate.path);
    if (!absolute) { result(candidate.id, "OUT_OF_RESOLVER_SCOPE"); continue; }
    const source = program.getSourceFile(absolute);
    if (!source) { result(candidate.id, "UNRESOLVED"); continue; }
    const node = nodeAtRange(source, candidate, ts);
    if (!node) { result(candidate.id, "UNRESOLVED"); continue; }
    let symbol = unalias(symbolFor(node, candidate, checker, ts), checker, ts);
    if (!symbol) { result(candidate.id, "UNRESOLVED"); continue; }
    const declarations = (symbol.declarations || []).filter(Boolean);
    if (declarations.length !== 1) { result(candidate.id, declarations.length > 1 ? "AMBIGUOUS" : "UNRESOLVED"); continue; }
    const declaration = declarations[0];
    const targetSource = declaration.getSourceFile();
    const targetAbsolute = path.resolve(targetSource.fileName);
    let targetPath = null;
    for (const [relativePath, candidateAbsolute] of universe) if (candidateAbsolute === targetAbsolute) targetPath = relativePath;
    if (!targetPath) { result(candidate.id, "OUT_OF_RESOLVER_SCOPE"); continue; }
    const [start, end] = byteRange(targetSource.text, declaration, ts);
    if (end <= start) { result(candidate.id, "UNRESOLVED"); continue; }
    result(candidate.id, "RESOLVED_UNIQUE", { target_path: targetPath, target_start_byte: start, target_end_byte: end });
  }
}

const raw = fs.readFileSync(0, "utf8");
try { if (!raw.endsWith("\n") || raw.trim().split("\n").length !== 1) throw new Error("exactly one JSON request line is required"); main(JSON.parse(raw)); } catch (error) { fail(error instanceof Error ? error.message : "resolver failure"); }
