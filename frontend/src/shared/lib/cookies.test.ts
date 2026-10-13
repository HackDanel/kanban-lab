import { describe, expect, it } from 'vitest'
import { getCookie } from './cookies'

describe('getCookie', () => {
  it('возвращает null, если cookie нет', () => {
    expect(getCookie('csrf_token')).toBeNull()
  })

  it('находит cookie среди нескольких и декодирует значение', () => {
    document.cookie = 'other=1'
    document.cookie = 'csrf_token=a%20b'
    expect(getCookie('csrf_token')).toBe('a b')
  })

  it('не путает cookie с общим суффиксом имени', () => {
    document.cookie = 'x_csrf_token=wrong'
    expect(getCookie('csrf_token')).toBeNull()
  })
})
