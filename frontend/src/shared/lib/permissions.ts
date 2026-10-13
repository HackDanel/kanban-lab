import type { Project, Role, User } from '../api/types'

const MANAGING_ROLES: readonly Role[] = ['admin', 'manager']

/** Создание проектов, досок, колонок и меток. */
export function canManage(user: User | null): boolean {
  return user !== null && MANAGING_ROLES.includes(user.role)
}

/** Изменение и удаление проекта: администратор или менеджер-владелец. */
export function canEditProject(user: User | null, project: Project): boolean {
  if (user === null) return false
  return user.role === 'admin' || (user.role === 'manager' && project.ownerId === user.id)
}

export function isAdmin(user: User | null): boolean {
  return user?.role === 'admin'
}

export const ROLE_LABELS: Record<Role, string> = {
  admin: 'Администратор',
  manager: 'Менеджер',
  developer: 'Разработчик',
}
