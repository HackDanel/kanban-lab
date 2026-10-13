import type { components } from './schema'

type Schemas = components['schemas']

export type ApiError = Schemas['Error']
export type Role = Schemas['Role']
export type Priority = Schemas['Priority']
export type User = Schemas['User']
export type AuthResponse = Schemas['AuthResponse']
export type RegisterRequest = Schemas['RegisterRequest']
export type LoginRequest = Schemas['LoginRequest']
export type Project = Schemas['Project']
export type ProjectInput = Schemas['ProjectInput']
export type ProjectPatch = Schemas['ProjectPatch']
