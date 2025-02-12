import Head from 'next/head'
import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR from 'swr'

import { useAdmin } from '../../lib/admin'

import Dashboard from '../../components/layouts/dashboard'
import Admin from '../../components/settings/admin'
import Account from '../../components/settings/account'
import Notification from '../../components/notification'

export default function Settings () {
  const router = useRouter()
  const { resetPassword } = router.query

  const { data: auth, error } = useSWR('/api/users/self')
  const { admin, loading: adminLoading } = useAdmin()

  const [showNotification, setshowNotification] = useState(resetPassword === 'success')

  const loading = adminLoading || (!auth && !error)
  const hasInfraProvider = auth?.providerNames.includes('infra')

  return (
    <>
      <Head>
        <title>Settings - Infra</title>
      </Head>
      {!loading && (
        <div className='flex-1 flex flex-col space-y-8 mt-6 mb-4'>
          <h1 className='text-xs mb-6 font-bold'>Settings</h1>
          {hasInfraProvider && <Account />}
          {admin && <Admin />}
          {resetPassword && <Notification show={showNotification} setShow={setshowNotification} text='Password Successfully Reset' />}
        </div>
      )}
    </>
  )
}

Settings.layout = function (page) {
  return (
    <Dashboard>
      {page}
    </Dashboard>
  )
}
