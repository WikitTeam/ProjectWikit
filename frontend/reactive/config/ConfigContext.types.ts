import { UserData } from '~api/user'

export interface IConfigContext {
  user: UserData
  site?: { title: string; systemIcon?: string }
  reviewsTickets?: boolean
}
