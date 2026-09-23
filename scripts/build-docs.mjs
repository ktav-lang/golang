#!/usr/bin/env node
// Rebuilds this repository's published Markdown documents from
// root-docs/, using @ktav-lang/polydoc. Run with --check for a
// CI-friendly, read-only verification instead of regenerating the files.

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { configure, buildRootDocs } from '@ktav-lang/polydoc';

const LANGS = ['en', 'ru', 'zh'];
configure({
  langs: LANGS,
  rootDocuments: ['README', 'CHANGELOG', 'CONTRIBUTING', 'SECURITY', 'EXAMPLES'],
});

// Unlike ktav-lang/rust, this repository keeps its ru/zh translations
// under docs/, keeps CONTRIBUTING/SECURITY entirely in docs/, and has an
// examples/README.md — legacy paths other tooling and cross-links depend
// on. polydoc 0.1.0 can only address root documents next to the repo root
// (rootOutputName is not configurable, and configure() rejects path
// separators in output names), so the write/check below mirrors
// polydoc's writeRootDocs/checkRootDocs against these mapped paths while
// polydoc keeps doing all assembly and source validation.
const OUTPUT_PATHS = {
  README: { en: 'README.md', ru: 'docs/README.ru.md', zh: 'docs/README.zh.md' },
  CHANGELOG: { en: 'CHANGELOG.md', ru: 'docs/CHANGELOG.ru.md', zh: 'docs/CHANGELOG.zh.md' },
  CONTRIBUTING: { en: 'docs/CONTRIBUTING.md', ru: 'docs/CONTRIBUTING.ru.md', zh: 'docs/CONTRIBUTING.zh.md' },
  SECURITY: { en: 'docs/SECURITY.md', ru: 'docs/SECURITY.ru.md', zh: 'docs/SECURITY.zh.md' },
  EXAMPLES: { en: 'examples/README.md', ru: 'examples/README.ru.md', zh: 'examples/README.zh.md' },
};

// SECURITY.md's supported-version row uses @@MINOR_LINE@@ so it can never
// quietly fall behind the module's actual version the way a hand-written
// "0.1.x" once did. The Go module has no version field of its own; the
// embedded LibVersion constant is the release's source of truth — the
// release tag and the native-library asset name both follow it.
function readModuleVersion(root) {
  const go = fs.readFileSync(path.join(root, 'internal', 'native', 'loader.go'), 'utf8');
  const match = go.match(/^const LibVersion = "([^"]+)"/mu);
  if (!match) throw new Error('internal/native/loader.go: no LibVersion constant found');
  return match[1];
}

// Per-unit validation proves every meaning has every language. It does
// NOT prove the languages describe the same DOCUMENT: a heading demoted
// from ## to ### in one translation, or an extra heading in another,
// passes unit validation untouched. This check (as in ktav-lang/rust)
// is what catches that class of drift.
function headingSkeleton(markdown) {
  const levels = [];
  let fenceChar = null;
  let fenceLen = 0;
  for (const line of markdown.split('\n')) {
    const fence = line.match(/^\s{0,3}(`{3,}|~{3,})/u);
    if (fence) {
      const char = fence[1][0];
      const len = fence[1].length;
      if (fenceChar === null) { fenceChar = char; fenceLen = len; }
      else if (char === fenceChar && len >= fenceLen) { fenceChar = null; }
      continue;
    }
    if (fenceChar !== null) continue;
    const heading = line.match(/^(#{1,6})\s+\S/u);
    if (heading) levels.push(heading[1].length);
  }
  return levels;
}

function structuralProblems(label, perLang) {
  const [reference, ...others] = LANGS;
  const base = headingSkeleton(perLang.get(reference).toString('utf8'));
  const problems = [];
  for (const lang of others) {
    const other = headingSkeleton(perLang.get(lang).toString('utf8'));
    if (other.length !== base.length) {
      problems.push(`${label}: ${lang} has ${other.length} heading(s) but ${reference} has ` +
        `${base.length} — the translations describe different documents`);
      continue;
    }
    const at = base.findIndex((level, i) => level !== other[i]);
    if (at !== -1) {
      problems.push(`${label}: heading #${at + 1} is level ${other[at]} in ${lang} but ` +
        `level ${base[at]} in ${reference}`);
    }
  }
  return problems;
}

function writeMappedRootDocs(root, docs) {
  for (const [doc, perLang] of docs) {
    for (const lang of LANGS) {
      fs.writeFileSync(path.join(root, OUTPUT_PATHS[doc][lang]), perLang.get(lang));
    }
  }
}

function checkMappedRootDocs(root, docs) {
  const problems = [];
  for (const [doc, perLang] of docs) {
    for (const lang of LANGS) {
      const rel = OUTPUT_PATHS[doc][lang];
      const expected = perLang.get(lang);
      let actual;
      try {
        actual = fs.readFileSync(path.join(root, rel));
      } catch {
        problems.push(`${rel} is missing or unreadable; it is generated from root-docs/${doc}/`);
        continue;
      }
      if (actual.equals(expected)) continue;
      let off = 0;
      const min = Math.min(actual.length, expected.length);
      while (off < min && actual[off] === expected[off]) off++;
      const line = expected.subarray(0, off).toString('utf8').split('\n').length;
      problems.push(
        `${rel} differs from what root-docs/${doc}/ generates, first at byte ${off} ` +
        `(line ${line}); edit the unit source, never the generated file`);
    }
  }
  return problems;
}

function usage() {
  process.stderr.write(
    'usage: node scripts/build-docs.mjs [--check]\n' +
    '  (no args)  regenerate the published Markdown documents from root-docs/\n' +
    '  --check    verify the generated files match root-docs/ without writing\n'
  );
}

function cli() {
  const scriptDir = path.dirname(fileURLToPath(import.meta.url));
  const root = path.resolve(scriptDir, '..');

  const args = process.argv.slice(2);
  if (args.includes('-h') || args.includes('--help')) { usage(); process.exit(0); }
  if (args.length > 1 || (args.length === 1 && args[0] !== '--check')) {
    usage();
    process.exit(1);
  }
  const checkMode = args[0] === '--check';

  let docs;
  try {
    const version = readModuleVersion(root);
    docs = buildRootDocs(root, { release: { version, released: '' } });
    for (const [name, perLang] of docs) {
      const problems = structuralProblems(name, perLang);
      if (problems.length > 0) {
        throw new Error(problems.join('\n'));
      }
    }
  } catch (e) {
    process.stderr.write(`build-docs: ${e.message}\n`);
    process.exit(1);
  }

  if (!checkMode) {
    writeMappedRootDocs(root, docs);
    process.stdout.write(`build-docs: assembled ${docs.size} document(s) from root-docs/\n`);
    process.exit(0);
  }

  const problems = checkMappedRootDocs(root, docs);
  if (problems.length > 0) {
    for (const problem of problems) {
      process.stderr.write(`build-docs --check: ${problem}\n`);
    }
    process.exit(1);
  }
  process.exit(0);
}

const isMain = process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;
if (isMain) cli();
