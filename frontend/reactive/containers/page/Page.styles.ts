import styled, { createGlobalStyle } from 'styled-components'

export const RootStyles = createGlobalStyle`
  *, *::before, *::after {
    box-sizing: border-box;
  }

  html, body {
    background: ${({ theme }) => theme.windowBackground};
    color: ${({ theme }) => theme.foreground};
    font-family: 'Inter Tight', 'Inter', -apple-system, BlinkMacSystemFont,
      'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
    font-size: 15px;
    line-height: 1.55;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;
    margin: 0;
    padding: 0;
    min-height: 100vh;
  }

  :where(#reactive-root) a:not([class]) {
    color: ${({ theme }) => theme.accent};
  }

  :where(#reactive-root) a:not([class]):hover {
    color: ${({ theme }) => theme.primary};
  }

  #reactive-root :focus-visible {
    outline: 2px solid ${({ theme }) => theme.accent};
    outline-offset: 2px;
  }

  ::selection {
    background: ${({ theme }) => theme.uiSelection};
    color: ${({ theme }) => theme.uiSelectionForeground};
  }
`

export const Container = styled.div`
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: ${({ theme }) => theme.windowBackground};
`

export const MainContainer = styled.div`
  flex: 1;
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
  padding: 32px 24px 96px;
  background: ${({ theme }) => theme.uiBackground};
  border-left: 1px solid ${({ theme }) => theme.uiBorder};
  border-right: 1px solid ${({ theme }) => theme.uiBorder};

  @media (max-width: 700px) {
    padding: 20px 16px 20px;
    border-left: none;
    border-right: none;
  }
`
