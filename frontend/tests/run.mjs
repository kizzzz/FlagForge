import { build } from 'esbuild'
import { spawnSync } from 'node:child_process'
import { mkdir } from 'node:fs/promises'

await mkdir('.test-build', { recursive: true })
await build({ entryPoints: ['tests/regressions.test.jsx'], outfile: '.test-build/regressions.test.cjs', bundle: true, platform: 'node', format: 'cjs', packages: 'external', loader: { '.css': 'empty' }, jsx: 'automatic' })
const result = spawnSync(process.execPath, ['--import', './tests/setup.mjs', '--test', '--test-force-exit', ...process.argv.slice(2), '.test-build/regressions.test.cjs'], { stdio: 'inherit' })
process.exit(result.status ?? 1)
