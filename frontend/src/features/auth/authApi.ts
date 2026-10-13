import { baseApi, csrfHeaders } from '../../shared/api/baseApi'
import type { AuthResponse, LoginRequest, RegisterRequest, User } from '../../shared/api/types'
import { credentialsReceived, loggedOut } from './authSlice'

export const authApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    register: build.mutation<User, RegisterRequest>({
      query: (body) => ({ url: '/auth/register', method: 'POST', body }),
    }),
    login: build.mutation<AuthResponse, LoginRequest>({
      query: (body) => ({ url: '/auth/login', method: 'POST', body }),
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled
          dispatch(credentialsReceived(data))
        } catch {
          // ошибку показывает форма входа
        }
      },
    }),
    // Восстановление сессии при загрузке страницы по refresh-cookie.
    restoreSession: build.mutation<AuthResponse, void>({
      query: () => ({ url: '/auth/refresh', method: 'POST', headers: csrfHeaders() }),
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled
          dispatch(credentialsReceived(data))
        } catch {
          dispatch(loggedOut())
        }
      },
    }),
    logout: build.mutation<void, void>({
      query: () => ({ url: '/auth/logout', method: 'POST', headers: csrfHeaders() }),
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          await queryFulfilled
        } catch {
          // сервер недоступен: локальную сессию всё равно завершаем
        }
        dispatch(loggedOut())
        dispatch(baseApi.util.resetApiState())
      },
    }),
  }),
})

export const {
  useRegisterMutation,
  useLoginMutation,
  useRestoreSessionMutation,
  useLogoutMutation,
} = authApi
