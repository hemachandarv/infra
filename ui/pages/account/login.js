import styled from 'styled-components'
import Link from 'next/link'
import Router from 'next/router'
import { useContext, useEffect } from 'react'

import AccountHeader from '../../components/AccountHeader'
import IdentitySourceBtn from '../../components/IdentitySourceBtn'

import AuthContext from '../../store/AuthContext'

const LoginContainer = styled.section`
  margin-left: auto;
  margin-right: auto;
  max-width: 24rem;
  padding-top: 2rem;

  display: grid;
  grid-template-rows: 1fr auto;
  min-height: 100%;
`

const Content = styled.div`
  & > *:not(:first-child) {
    padding-top: 1.5rem;
  }
`

const LoginIdentitySourceList = styled.div`
  margin-top: 2rem;
`

const HelpContainer = styled.div`
  margin-top: 3rem;
  font-weight: 100;
  font-size: 11px;
  line-height: 13px;
  max-width: 24rem;
  text-align: center;

  span {
    opacity: .5;
  }

  a {
    padding-left: .5rem;
    color: #93DEFF;
    text-decoration: none;

    :hover {
      opacity: .95;
    }
  }
`

export const readyToRedirect = async () => {
  await Router.push({
    pathname: '/'
  }, undefined, { shallow: true })
}

const Login = () => {
  const { providers, authReady, hasRedirected } = useContext(AuthContext)

  const getProviderType = (url) => {
    const tempURL = url
    return tempURL.replace(/^https?:\/\//, '').split('/')[0].split('.').reverse()[1]
  }

  const providerWithType = providers.map((item) => {
    const type = getProviderType(item.url)
    return { ...item, type }
  })

  useEffect(() => {
    if (authReady) {
      readyToRedirect()
    }
  }, [])

  return (
    <LoginContainer>
      <Content>
        {hasRedirected
          ? (<></>)
          : (
            <>
              <AccountHeader
                header='Login to Infra'
                subheader='Securely manage access to your infrastructure. Take a moment to create your account and start managing access today.'
              />
              <LoginIdentitySourceList>
                <IdentitySourceBtn providers={providerWithType} />
              </LoginIdentitySourceList>
              <HelpContainer>
                <span>Having trouble logging in?</span>
                <Link href='/account/register'>
                  <a>Use API Access Key</a>
                </Link>
              </HelpContainer>
            </>)}
      </Content>
    </LoginContainer>
  )
}

export default Login