import { describe, expect, it } from 'vitest'
import { getErrorCode, getErrorMessage, getFieldErrors } from './errors'

const validation = {
  status: 400,
  data: {
    code: 'VALIDATION_ERROR',
    message: 'Некорректные данные',
    details: [
      { field: 'title', message: 'Название обязательно' },
      { field: 'title', message: 'второе сообщение' },
      { field: 'priority', message: 'Недопустимое значение' },
    ],
  },
}

describe('ошибки API', () => {
  it('читает код и сообщение из ответа API', () => {
    expect(getErrorCode(validation)).toBe('VALIDATION_ERROR')
    expect(getErrorMessage(validation)).toBe('Некорректные данные')
  })

  it('собирает ошибки по полям, оставляя первое сообщение для поля', () => {
    expect(getFieldErrors(validation)).toEqual({
      title: 'Название обязательно',
      priority: 'Недопустимое значение',
    })
  })

  it.each([
    ['сетевой сбой', { status: 'FETCH_ERROR', error: 'Failed to fetch' }],
    ['ответ не в формате API', { status: 502, data: '<html>Bad Gateway</html>' }],
    ['объект без кода', { status: 500, data: { message: 'x' } }],
    ['не объект', 'boom'],
    ['null', null],
  ])('даёт запасное сообщение: %s', (_name, error) => {
    expect(getErrorCode(error)).toBeNull()
    expect(getErrorMessage(error)).toMatch(/Не удалось выполнить запрос/)
    expect(getFieldErrors(error)).toEqual({})
  })
})
