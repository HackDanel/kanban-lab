import type { RegisterRequest } from '../../shared/api/types'

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
export const MIN_PASSWORD_LENGTH = 8
// bcrypt учитывает только первые 72 байта пароля
export const MAX_PASSWORD_BYTES = 72

export type RegisterErrors = Partial<Record<keyof RegisterRequest, string>>

export function validateRegistration(values: RegisterRequest): RegisterErrors {
  const errors: RegisterErrors = {}
  if (values.name.trim() === '') errors.name = 'Имя обязательно'
  if (!EMAIL.test(values.email.trim())) errors.email = 'Укажите корректный email'
  if (values.password.length < MIN_PASSWORD_LENGTH) {
    errors.password = `Пароль должен быть не короче ${MIN_PASSWORD_LENGTH} символов`
  } else if (new TextEncoder().encode(values.password).length > MAX_PASSWORD_BYTES) {
    errors.password = 'Пароль слишком длинный'
  }
  return errors
}
