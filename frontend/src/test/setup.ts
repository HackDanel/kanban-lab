import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from './server'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => {
  cleanup()
  server.resetHandlers()
  for (const part of document.cookie.split('; ')) {
    document.cookie = `${part.split('=')[0]}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`
  }
})
afterAll(() => server.close())
