import { setupServer } from 'msw/node'

export const server = setupServer()

/** Путь API для обработчиков MSW. */
export const api = (path: string) => `*/api/v1${path}`
