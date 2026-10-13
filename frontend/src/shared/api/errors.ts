import type { ApiError } from './types'

const FALLBACK = 'Не удалось выполнить запрос. Попробуйте ещё раз.'

function asApiError(error: unknown): ApiError | null {
  if (typeof error !== 'object' || error === null || !('data' in error)) return null
  const data = (error as { data: unknown }).data
  if (typeof data !== 'object' || data === null) return null
  const { code, message } = data as Record<string, unknown>
  return typeof code === 'string' && typeof message === 'string' ? (data as ApiError) : null
}

/** Код ошибки API (`VALIDATION_ERROR`, `WIP_LIMIT_EXCEEDED`, ...) или null для сетевых сбоев. */
export function getErrorCode(error: unknown): string | null {
  return asApiError(error)?.code ?? null
}

export function getErrorMessage(error: unknown): string {
  return asApiError(error)?.message ?? FALLBACK
}

/** Ошибки валидации по полям: `{ title: 'Название обязательно' }`. */
export function getFieldErrors(error: unknown): Record<string, string> {
  const result: Record<string, string> = {}
  for (const detail of asApiError(error)?.details ?? []) {
    result[detail.field] ??= detail.message
  }
  return result
}
