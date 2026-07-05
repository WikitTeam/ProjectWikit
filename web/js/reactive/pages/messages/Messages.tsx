import * as React from 'react'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ProfilePage } from '~reactive/containers/page'
import { lookupUser } from '~api/user'
import { Paths } from '~reactive/paths'
import ConversationList from './ConversationList'
import ConversationView from './ConversationView'
import * as Styled from './Messages.styles'

const Messages: React.FC = () => {
  const params = useParams<{ user_id?: string }>()
  const partnerId = params.user_id ? parseInt(params.user_id, 10) : null
  const [refreshToken, setRefreshToken] = useState<number>(0)
  const [searchInput, setSearchInput] = useState<string>('')
  const [searchError, setSearchError] = useState<string | null>(null)
  const [searching, setSearching] = useState<boolean>(false)
  const navigate = useNavigate()

  const bumpList = () => setRefreshToken(v => v + 1)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    const name = searchInput.trim()
    if (!name || searching) return

    setSearchError(null)
    setSearching(true)
    try {
      const user = await lookupUser(name)
      if (!user.id) {
        setSearchError('未找到该用户')
        return
      }
      setSearchInput('')
      navigate(`${Paths.messages}/${user.id}`)
    } catch (err: any) {
      setSearchError(err?.error || '未找到该用户')
    } finally {
      setSearching(false)
    }
  }

  return (
    <ProfilePage>
      <Styled.Layout>
        <Styled.Sidebar hasSelection={partnerId !== null}>
          <Styled.SidebarSearch onSubmit={handleSearch}>
            <Styled.SearchInput
              type="text"
              value={searchInput}
              onChange={e => setSearchInput(e.target.value)}
              placeholder="输入用户名查找并发信…"
            />
            <Styled.SearchButton type="submit" disabled={searching || !searchInput.trim()}>
              {searching ? '查找中' : '查找'}
            </Styled.SearchButton>
          </Styled.SidebarSearch>
          {searchError && <Styled.SearchError>{searchError}</Styled.SearchError>}
          <ConversationList selectedPartnerId={partnerId} refreshToken={refreshToken} />
        </Styled.Sidebar>
        <Styled.Main>
          {partnerId !== null && !Number.isNaN(partnerId) ? (
            <ConversationView partnerId={partnerId} onMessageSent={bumpList} />
          ) : (
            <Styled.EmptyState>请选择一个会话查看，或前往任意用户资料页发起私信。</Styled.EmptyState>
          )}
        </Styled.Main>
      </Styled.Layout>
    </ProfilePage>
  )
}

export default Messages
