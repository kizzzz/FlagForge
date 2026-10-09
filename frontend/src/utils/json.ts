import type { Rule } from '../api/client'
import { diagnoseJSONSyntax } from './jsonSyntaxOffset'
export const isObject = (v: unknown): v is Record<string, unknown> => Boolean(v) && typeof v === 'object' && !Array.isArray(v)
export const isScalar = (v: unknown) => typeof v === 'number' ? Number.isFinite(v) : typeof v === 'string' || typeof v === 'boolean'
export function hasNonFinite(value: unknown): boolean {
  if (typeof value === 'number') return !Number.isFinite(value)
  if (Array.isArray(value)) return value.some(hasNonFinite)
  return isObject(value) && Object.values(value).some(hasNonFinite)
}
export function parseJSON(text: string): unknown {
  let value: unknown
  try { value = JSON.parse(text) } catch (error) {
    const message = error instanceof Error ? error.message : 'JSON 语法错误'
    // Match native diagnostic suffixes, never words such as "position 0" echoed from user input.
    const nativePosition = message.match(/\bJSON at position\s+(\d+)(?:\s+\(line\s+\d+\s+column\s+\d+\))?$/i)?.[1]
    const nativeLine = message.match(/\bat line\s+(\d+)\s+column\s+\d+ of the JSON data$/i)?.[1]
    const position = nativePosition === undefined ? undefined : Number(nativePosition)
    const reliablePosition = position !== undefined && position >= 0 && position <= text.length
    let line = reliablePosition ? text.slice(0, position).split('\n').length : nativeLine === undefined ? undefined : Number(nativeLine)
    if (line === undefined || line < 1 || line > text.split('\n').length) {
      const offset = diagnoseJSONSyntax(text)
      line = offset === undefined ? undefined : text.slice(0, offset).split('\n').length
    }
    throw new Error(`JSON 语法错误${line === undefined ? '（解析器未提供精确行号）' : `（第 ${line} 行）`}：${message}`)
  }
  if (hasNonFinite(value)) throw new Error('数字超出有限 JSON 数值范围，请使用可序列化的有限数字；不会将溢出数字静默改为 null')
  return value
}
export function parseObject(text: string) {
  const value = parseJSON(text)
  if (!isObject(value)) throw new Error('必须是 JSON object，不能是数组、null 或标量')
  return value
}
export function parseExpected(text: string, operator: Rule['operator']): Rule['expected_value'] {
  const value = parseJSON(text)
  if (operator === 'equals' ? !isScalar(value) : !Array.isArray(value) || !value.length || !value.every(isScalar)) {
    throw new Error(operator === 'equals' ? 'equals 必须是 string / number / boolean 标量' : 'in 必须是非空数组，且每个元素都是 string / number / boolean 标量')
  }
  return value as Rule['expected_value']
}
export function parseBatch(text: string) {
  const lines: number[] = [], values: Record<string, unknown>[] = []
  text.split('\n').forEach((row, index) => {
    if (!row.trim()) return
    try { values.push(parseObject(row)); lines.push(index + 1) } catch (error) { throw new Error(`第 ${index + 1} 行：${error instanceof Error ? error.message : '输入无效'}`) }
  })
  if (!values.length) throw new Error('请至少输入一行 JSON object')
  if (values.length > 100) throw new Error('最多支持 100 组用户属性')
  // Server index is zero-based after empty lines are removed; lines[index] is the original one-based editor line.
  return { values, lines }
}
export function validationError(fn: () => unknown) { try { fn(); return '' } catch (error) { return error instanceof Error ? error.message : '输入无效' } }
export const shellQuote = (text: string) => `'${text.split("'").join("'\\''")}'`
export function evaluationCurl(input: unknown, origin: string) { return `curl ${shellQuote(`${origin}/api/v1/evaluate`)} -H 'Content-Type: application/json' --data-raw ${shellQuote(JSON.stringify(input))}` }
