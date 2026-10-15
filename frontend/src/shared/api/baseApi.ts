import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react'
import type { BaseQueryFn, FetchArgs, FetchBaseQueryError } from '@reduxjs/toolkit/query'
import { credentialsReceived, loggedOut } from '../../features/auth/authSlice'
import type { AuthState } from '../../features/auth/authSlice'
import { getCookie } from '../lib/cookies'
import type { AuthResponse } from './types'

export const CSRF_COOKIE = 'csrf_token'
export const CSRF_HEADER = 'X-CSRF-Token'

const rawBaseQuery = fetchBaseQuery({
  baseUrl: new URL('/api/v1', window.location.origin).href,
  credentials: 'include',
  prepareHeaders: (headers, { getState }) => {
    const token = (getState() as { auth: AuthState }).auth.accessToken
    if (token) headers.set('Authorization', `Bearer ${token}`)
    return headers
  },
})

type BaseQuery = BaseQueryFn<string | FetchArgs, unknown, FetchBaseQueryError>

export function csrfHeaders(): Record<string, string> {
  const token = getCookie(CSRF_COOKIE)
  return token ? { [CSRF_HEADER]: token } : {}
}

// Один запрос на обновление токена на все параллельно упавшие с 401 запросы.
let refreshing: Promise<boolean> | null = null

const refreshSession: (...args: Parameters<BaseQuery>) => Promise<boolean> = async (
  _args,
  api,
  extraOptions,
) => {
  const result = await rawBaseQuery(
    { url: '/auth/refresh', method: 'POST', headers: csrfHeaders() },
    api,
    extraOptions,
  )
  if (result.data) {
    api.dispatch(credentialsReceived(result.data as AuthResponse))
    return true
  }
  api.dispatch(loggedOut())
  return false
}

function urlOf(args: string | FetchArgs): string {
  return typeof args === 'string' ? args : args.url
}

const baseQueryWithReauth: BaseQuery = async (args, api, extraOptions) => {
  const result = await rawBaseQuery(args, api, extraOptions)
  if (result.error?.status !== 401 || urlOf(args).startsWith('/auth/')) return result

  refreshing ??= refreshSession(args, api, extraOptions).finally(() => {
    refreshing = null
  })
  return (await refreshing) ? rawBaseQuery(args, api, extraOptions) : result
}

export const baseApi = createApi({
  reducerPath: 'api',
  baseQuery: baseQueryWithReauth,
  tagTypes: [
    'Project',
    'Member',
    'Label',
    'Board',
    'Cards',
    'Comment',
    'Attachment',
    'Activity',
    'User',
  ],
  endpoints: () => ({}),
})
