import { describe, expect, it } from 'vitest'
import { makeStore } from '../../app/store'
import { makeAuth, makeUser } from '../../test/fixtures'
import {
  credentialsReceived,
  loggedOut,
  selectAccessToken,
  selectAuthStatus,
  selectCurrentUser,
} from './authSlice'

describe('authSlice', () => {
  it('до проверки сессии статус неизвестен', () => {
    const state = makeStore().getState()
    expect(selectAuthStatus(state)).toBe('unknown')
    expect(selectCurrentUser(state)).toBeNull()
    expect(selectAccessToken(state)).toBeNull()
  })

  it('сохраняет пользователя и токен после входа', () => {
    const store = makeStore()
    const user = makeUser('manager')
    store.dispatch(credentialsReceived(makeAuth(user, 'token-1')))

    expect(selectAuthStatus(store.getState())).toBe('authenticated')
    expect(selectCurrentUser(store.getState())).toEqual(user)
    expect(selectAccessToken(store.getState())).toBe('token-1')
  })

  it('очищает сессию при выходе', () => {
    const store = makeStore()
    store.dispatch(credentialsReceived(makeAuth()))
    store.dispatch(loggedOut())

    expect(selectAuthStatus(store.getState())).toBe('anonymous')
    expect(selectCurrentUser(store.getState())).toBeNull()
    expect(selectAccessToken(store.getState())).toBeNull()
  })
})
