import { describe, expect, it } from 'vitest'
import { validateRegistration } from './validation'

const valid = { name: 'Иван', email: 'ivan@example.com', password: 'password1' }

describe('validateRegistration', () => {
  it('принимает корректные данные', () => {
    expect(validateRegistration(valid)).toEqual({})
  })

  it('требует имя', () => {
    expect(validateRegistration({ ...valid, name: '   ' })).toEqual({ name: 'Имя обязательно' })
  })

  it.each(['', 'ivan', 'ivan@', 'ivan@example', 'i van@example.com'])(
    'отклоняет email «%s»',
    (email) => {
      expect(validateRegistration({ ...valid, email }).email).toBeDefined()
    },
  )

  it('проверяет границу минимальной длины пароля', () => {
    expect(validateRegistration({ ...valid, password: '1234567' }).password).toMatch(/не короче 8/)
    expect(validateRegistration({ ...valid, password: '12345678' }).password).toBeUndefined()
  })

  it('считает длину пароля в байтах, как bcrypt', () => {
    expect(validateRegistration({ ...valid, password: 'a'.repeat(72) }).password).toBeUndefined()
    expect(validateRegistration({ ...valid, password: 'a'.repeat(73) }).password).toBeDefined()
    // 37 кириллических символов — это 74 байта в UTF-8
    expect(validateRegistration({ ...valid, password: 'я'.repeat(37) }).password).toBe(
      'Пароль слишком длинный',
    )
  })
})
