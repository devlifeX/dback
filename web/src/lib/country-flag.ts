export function countryFlag(code?: string) {
  const c = (code ?? '').trim().toUpperCase()
  if (c.length !== 2) return ''
  return String.fromCodePoint(...c.split('').map((ch) => 0x1f1e6 - 65 + ch.charCodeAt(0)))
}

export function countryLabel(name: string, code?: string) {
  const flag = countryFlag(code)
  return flag ? `${flag} ${name}` : name
}
