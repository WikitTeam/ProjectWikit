import * as React from 'react'
import styled from 'styled-components'

interface StyledProps {
  color: string
}

type Props = Partial<StyledProps> & { className?: string }

const LoaderStyle = styled.div<StyledProps>`
  @keyframes pwikit-loader {
    0% {
      left: -35%;
      right: 100%;
    }
    100% {
      left: 100%;
      right: -35%;
    }
  }

  height: 2px;
  width: 100%;
  position: relative;
  overflow: hidden;

  div {
    position: absolute;
    top: 0;
    bottom: 0;
    background: ${props => props.color};
    animation: pwikit-loader 1.1s ease-in-out infinite;
    will-change: left, right;
  }
`

const Loader: React.FC<Props> = ({ className, color }) => {
  return (
    <LoaderStyle className={className} color={color || 'currentColor'}>
      <div />
    </LoaderStyle>
  )
}

export default Loader
