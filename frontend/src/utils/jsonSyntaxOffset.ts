type Frame = { kind: 'object' | 'array'; state: 'keyOrEnd' | 'key' | 'colon' | 'value' | 'valueOrEnd' | 'commaOrEnd' }

/** Locate the first grammar error only. Never constructs values or replaces JSON.parse. UTF-16 offset. */
export function diagnoseJSONSyntax(text: string): number | undefined {
  let index = 0
  const stack: Frame[] = []
  const fail = (): never => { throw index }
  const digit = () => index < text.length && text[index] >= '0' && text[index] <= '9'
  const whitespace = () => { while (index < text.length && /[ \t\r\n]/.test(text[index])) index++ }
  function string() {
    if (text[index] !== '"') fail()
    index++
    while (index < text.length) {
      if (text[index] === '"') { index++; return }
      if (text.charCodeAt(index) < 0x20) fail()
      if (text[index] === '\\') {
        index++
        if (index >= text.length) fail()
        if (text[index] === 'u') {
          index++
          for (let count = 0; count < 4; count++, index++) if (index >= text.length || !/[0-9a-fA-F]/.test(text[index])) fail()
          continue
        }
        if (!'"\\/bfnrt'.includes(text[index])) fail()
      }
      index++
    }
    fail()
  }
  function number() {
    if (text[index] === '-') index++
    if (!digit()) fail()
    if (text[index] === '0') index++
    else while (digit()) index++
    if (text[index] === '.') { index++; if (!digit()) fail(); while (digit()) index++ }
    if (text[index] === 'e' || text[index] === 'E') {
      index++
      if (text[index] === '+' || text[index] === '-') index++
      if (!digit()) fail()
      while (digit()) index++
    }
  }
  function value() {
    const token = text[index]
    if (token === '{' || token === '[') {
      index++; stack.push(token === '{' ? { kind: 'object', state: 'keyOrEnd' } : { kind: 'array', state: 'valueOrEnd' }); return
    }
    if (token === '"') { string(); return }
    if (token === '-' || digit()) { number(); return }
    const literal = token === 't' ? 'true' : token === 'f' ? 'false' : token === 'n' ? 'null' : undefined
    if (!literal) fail()
    for (const char of literal!) { if (text[index] !== char) fail(); index++ }
  }
  try {
    whitespace(); value()
    while (stack.length) {
      whitespace()
      const frame = stack[stack.length - 1]
      if (frame.state === 'keyOrEnd' || frame.state === 'key') {
        if (frame.state === 'keyOrEnd' && text[index] === '}') { index++; stack.pop(); continue }
        string(); frame.state = 'colon'
      } else if (frame.state === 'colon') {
        if (text[index] !== ':') fail()
        index++; frame.state = 'value'
      } else if (frame.state === 'valueOrEnd' || frame.state === 'value') {
        if (frame.state === 'valueOrEnd' && text[index] === ']') { index++; stack.pop(); continue }
        frame.state = 'commaOrEnd'; value()
      } else {
        const end = frame.kind === 'object' ? '}' : ']'
        if (text[index] === end) { index++; stack.pop() }
        else if (text[index] === ',') { index++; frame.state = frame.kind === 'object' ? 'key' : 'value' }
        else fail()
      }
    }
    whitespace()
    if (index !== text.length) fail()
    return undefined
  } catch (offset) {
    // Iterative container frames avoid recursion limits on deeply nested input.
    if (typeof offset === 'number') return offset
    throw offset
  }
}
