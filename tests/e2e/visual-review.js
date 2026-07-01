const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { repoRoot } = require('./helpers/state');

const expectedBaselineNames = [
  'setup',
  'inbox-equivalent-default-project',
  'today',
  'upcoming',
  'search',
  'quick-add',
  'conflicts',
  'settings'
].flatMap((name) => [
  `${name}-desktop.png`,
  `${name}-tablet.png`,
  `${name}-mobile.png`
]);

const args = parseArgs(process.argv.slice(2));
const browser = args.browser || 'chromium';
const baselineDir = args.dir
  ? path.resolve(repoRoot, args.dir)
  : path.join(repoRoot, 'test-results', 'e2e', browser, 'baselines');
const comparePath = args.compare ? path.resolve(repoRoot, args.compare) : '';
const outputPath = args.output
  ? path.resolve(repoRoot, args.output)
  : path.join(path.dirname(baselineDir), 'visual-review-manifest.json');

main();

function main() {
  const manifest = buildManifest(browser, baselineDir);
  const missing = expectedBaselineNames.filter((name) => !manifest.files.some((file) => file.name === name));
  if (missing.length > 0) {
    throw new Error(`missing expected visual review screenshots:\n${missing.map((name) => `- ${name}`).join('\n')}`);
  }

  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, `${JSON.stringify(manifest, null, 2)}\n`);
  console.log(`visual review manifest: ${path.relative(repoRoot, outputPath)}`);
  console.log(`screenshots: ${manifest.files.length}`);

  if (comparePath) {
    const previous = JSON.parse(fs.readFileSync(comparePath, 'utf8'));
    const diff = compareManifests(previous, manifest);
    printDiff(diff);
    if (diff.missing.length > 0 || diff.added.length > 0 || diff.changed.length > 0) {
      process.exitCode = 1;
    }
  }
}

function buildManifest(browserName, dir) {
  if (!fs.existsSync(dir)) {
    throw new Error(`baseline directory not found: ${path.relative(repoRoot, dir)}`);
  }

  const files = fs.readdirSync(dir)
    .filter((name) => name.endsWith('.png'))
    .sort((left, right) => left.localeCompare(right))
    .map((name) => {
      const filePath = path.join(dir, name);
      const data = fs.readFileSync(filePath);
      const dimensions = pngDimensions(data);
      return {
        name,
        path: path.relative(repoRoot, filePath),
        sha256: crypto.createHash('sha256').update(data).digest('hex'),
        bytes: data.length,
        width: dimensions.width,
        height: dimensions.height
      };
    });

  return {
    generated_at: new Date().toISOString(),
    commit: currentCommit(),
    browser: browserName,
    baseline_dir: path.relative(repoRoot, dir),
    expected: expectedBaselineNames,
    files
  };
}

function compareManifests(previous, current) {
  const previousFiles = new Map((previous.files || []).map((file) => [file.name, file]));
  const currentFiles = new Map((current.files || []).map((file) => [file.name, file]));
  const names = new Set([...previousFiles.keys(), ...currentFiles.keys()]);
  const diff = { changed: [], missing: [], added: [] };

  for (const name of [...names].sort((left, right) => left.localeCompare(right))) {
    const before = previousFiles.get(name);
    const after = currentFiles.get(name);
    if (!before && after) {
      diff.added.push(name);
      continue;
    }
    if (before && !after) {
      diff.missing.push(name);
      continue;
    }
    if (before.sha256 !== after.sha256 || before.width !== after.width || before.height !== after.height) {
      diff.changed.push({
        name,
        before: summarizeFile(before),
        after: summarizeFile(after)
      });
    }
  }

  return diff;
}

function printDiff(diff) {
  if (diff.missing.length === 0 && diff.added.length === 0 && diff.changed.length === 0) {
    console.log('visual review diff: no screenshot changes');
    return;
  }

  console.log('visual review diff: screenshot changes detected');
  for (const name of diff.missing) {
    console.log(`missing: ${name}`);
  }
  for (const name of diff.added) {
    console.log(`added: ${name}`);
  }
  for (const item of diff.changed) {
    console.log(`changed: ${item.name}`);
    console.log(`  before: ${item.before}`);
    console.log(`  after:  ${item.after}`);
  }
}

function summarizeFile(file) {
  return `${file.width}x${file.height} ${file.bytes} bytes ${String(file.sha256 || '').slice(0, 12)}`;
}

function pngDimensions(data) {
  const signature = '89504e470d0a1a0a';
  if (data.length < 24 || data.subarray(0, 8).toString('hex') !== signature) {
    throw new Error('expected a png screenshot');
  }
  return {
    width: data.readUInt32BE(16),
    height: data.readUInt32BE(20)
  };
}

function currentCommit() {
  try {
    return require('node:child_process')
      .execFileSync('git', ['rev-parse', '--short', 'HEAD'], { cwd: repoRoot, encoding: 'utf8' })
      .trim();
  } catch {
    return '';
  }
}

function parseArgs(rawArgs) {
  const parsed = {};
  for (let index = 0; index < rawArgs.length; index += 1) {
    const arg = rawArgs[index];
    if (!arg.startsWith('--')) {
      continue;
    }
    const [key, inlineValue] = arg.slice(2).split('=', 2);
    if (inlineValue !== undefined) {
      parsed[key] = inlineValue;
      continue;
    }
    const next = rawArgs[index + 1];
    if (next && !next.startsWith('--')) {
      parsed[key] = next;
      index += 1;
    } else {
      parsed[key] = 'true';
    }
  }
  return parsed;
}
