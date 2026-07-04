import * as React from 'react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { ProfilePage } from '~reactive/containers/page'
import ConversationList from './ConversationList'
import ConversationView from './ConversationView'
import * as Styled from './Messages.styles'

const Messages: React.FC = () => {
  const params = useParams<{ user_id?: string }>()
  const partnerId = params.user_id ? parseInt(params.user_id, 10) : null
  const [refreshToken, setRefreshToken] = useState<number>(0)

  const bumpList = () => setRefreshToken(v => v + 1)

  return (
    <ProfilePage>
      <Styled.Layout>
        <Styled.Sidebar hasSelection={partnerId !== null}>
          <ConversationList selectedPartnerId={partnerId} refreshToken={refreshToken} />
        </Styled.Sidebar>
        <Styled.Main>
          {partnerId !== null && !Number.isNaN(partnerId) ? (
            <ConversationView partnerId={partnerId} onMessageSent={bumpList} />
          ) : (
            <Styled.EmptyState>请选择左侧一个会话查看，或前往任意用户资料页发起私信。</Styled.EmptyState>
          )}
        </Styled.Main>
      </Styled.Layout>
    </ProfilePage>
  )
}

export default Messages
