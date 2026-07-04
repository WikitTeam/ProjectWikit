import { wFetch } from '../util/fetch-util'

export interface UserData {
  type: 'system' | 'anonymous' | 'bot' | 'normal' | 'wikidot'
  id?: number
  avatar?: string
  name: string
  username: string
  showAvatar: boolean
  editor?: boolean
  staff?: boolean
  admin?: boolean
  roles?: string
}

export function fetchAllUsers(): Promise<UserData[]> {
  return wFetch<UserData[]>('/api/users')
}

export function lookupUser(username: string): Promise<UserData> {
  return wFetch<UserData>(`/api/users/lookup?username=${encodeURIComponent(username)}`)
}

export interface AdminSusUser {
  user: {
    id: number
    name: string
  }
  ip: string
}

export function fetchAdminSusUsers(): Promise<AdminSusUser[]> {
  return wFetch<AdminSusUser[]>('/api/admin/sus')
}
