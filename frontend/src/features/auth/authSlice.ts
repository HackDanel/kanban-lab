import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { AuthResponse, User } from '../../shared/api/types'

export type AuthStatus = 'unknown' | 'authenticated' | 'anonymous'

export interface AuthState {
  status: AuthStatus
  accessToken: string | null
  user: User | null
}

const initialState: AuthState = { status: 'unknown', accessToken: null, user: null }

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    credentialsReceived(state, action: PayloadAction<AuthResponse>) {
      state.status = 'authenticated'
      state.accessToken = action.payload.accessToken
      state.user = action.payload.user
    },
    loggedOut(state) {
      state.status = 'anonymous'
      state.accessToken = null
      state.user = null
    },
  },
  selectors: {
    selectAuthStatus: (state) => state.status,
    selectCurrentUser: (state) => state.user,
    selectAccessToken: (state) => state.accessToken,
  },
})

export const { credentialsReceived, loggedOut } = authSlice.actions
export const { selectAuthStatus, selectCurrentUser, selectAccessToken } = authSlice.selectors
export const authReducer = authSlice.reducer
export const AUTH_SLICE = authSlice.name
