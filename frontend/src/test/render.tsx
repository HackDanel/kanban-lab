import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { Provider } from 'react-redux'
import { MemoryRouter } from 'react-router'
import type { InitialEntry } from 'react-router'
import { makeStore } from '../app/store'
import type { AppStore, RootState } from '../app/store'
import type { User } from '../shared/api/types'

interface Options {
  route?: InitialEntry
  preloadedState?: Partial<RootState>
  store?: AppStore
}

export function renderWithProviders(ui: ReactElement, options: Options = {}) {
  const store = options.store ?? makeStore(options.preloadedState)
  const view = render(
    <Provider store={store}>
      <MemoryRouter initialEntries={[options.route ?? '/']}>{ui}</MemoryRouter>
    </Provider>,
  )
  return { store, ...view }
}

export function signedIn(user: User, accessToken = 'access-1'): Partial<RootState> {
  return { auth: { status: 'authenticated', accessToken, user } }
}

export const anonymous: Partial<RootState> = {
  auth: { status: 'anonymous', accessToken: null, user: null },
}
