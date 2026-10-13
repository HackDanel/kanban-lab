import type { ProjectInput } from '../../shared/api/types'

const KEY = /^[A-Z][A-Z0-9]{1,9}$/

export type ProjectErrors = Partial<Record<keyof ProjectInput, string>>

export function validateProject(values: ProjectInput): ProjectErrors {
  const errors: ProjectErrors = {}
  if (!KEY.test(values.key)) {
    errors.key = 'От 2 до 10 латинских заглавных букв и цифр, первая — буква'
  }
  if (values.name.trim() === '') errors.name = 'Название обязательно'
  return errors
}
