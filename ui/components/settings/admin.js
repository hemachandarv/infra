import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import { PlusIcon } from '@heroicons/react/outline'

import InputDropdown from '../input'
import DeleteModal from '../modals/delete'
import ErrorMessage from '../error-message'

function Grant ({ id, userID, grants }) {
  if (!id || !userID) {
    return null
  }

  const { data: user } = useSWR(`/api/users/${userID}`, { fallbackData: { name: '', kind: '' } })
  const { data: auth } = useSWR('/api/users/self')
  const { mutate } = useSWRConfig()
  const [open, setOpen] = useState(false)

  const isSelf = userID === auth.id

  return (
    <div className='flex group'>
      <div className='flex flex-1 items-center space-x-4 py-1'>
        <div className='border border-violet-300/20 flex-none flex items-center justify-center w-8 h-8 rounded-lg'>
          <div className='relative text-center border border-violet-300/40 flex-none text-gray-500 flex justify-center items-center text-sm w-6 h-6 rounded-[4px]'>
            <span className='absolute inset-x-0 -mt-0.5'>{user?.name?.[0]}</span>
          </div>
        </div>
        <div className='flex flex-col leading-tight'>
          <div className='text-2xs leading-none'>{user.name}</div>
        </div>
      </div>

      <div className='opacity-0 group-hover:opacity-100 flex justify-end text-right'>
        {!isSelf && <button onClick={() => setOpen(true)} className='flex-none p-2 -mr-2 cursor-pointer text-2xs text-gray-500 hover:text-violet-100'>Revoke</button>}
        <DeleteModal
          open={open}
          setOpen={setOpen}
          primaryButtonText='Revoke'
          onSubmit={() => {
            mutate('/api/grants?resource=infra&privilege=admin', async ({ items: grants } = { items: [] }) => {
              await fetch(`/api/grants/${id}`, { method: 'DELETE' })
              return { items: grants?.filter(g => g?.id !== id) }
            }, { optimisticData: { items: grants.filter(g => g?.id !== id) } })

            setOpen(false)
          }}
          title='Revoke Admin'
          message={(<>Are you sure you want to revoke admin access for <span className='font-bold text-white'>{user.name}</span>?</>)}
        />
      </div>
    </div>
  )
}

export default function () {
  const { data: { items: grants } = {} } = useSWR(() => '/api/grants?resource=infra&privilege=admin', { fallbackData: [] })
  const { data: auth } = useSWR('/api/users/self')
  const { mutate } = useSWRConfig()

  const [name, setName] = useState('')
  const [error, setError] = useState('')

  const userGrants = grants?.filter(g => g.user)
  const sortedGrants = userGrants && [...userGrants?.filter(g => g.user === auth.id) || [], ...userGrants?.filter(g => g.user !== auth.id).sort((a, b) => a.user.localeCompare(b.user))]

  const grantAdminAccess = id => {
    fetch('/api/grants', {
      method: 'POST',
      body: JSON.stringify({ user: id, resource: 'infra', privilege: 'admin' })
    })
      .then(() => {
        mutate('/api/grants?resource=infra&privilege=admin')
        setName('')
      }).catch((e) => setError(e.message || 'something went wrong, please try again later.'))
  }

  const handleInputChange = (value) => {
    setName(value)
    setError('')
  }

  const handleKeyDownEvent = (key) => {
    if (key === 'Enter' && name.length > 0) {
      handleAddAdmin()
    }
  }

  const handleAddAdmin = () => {
    setError('')

    fetch(`/api/users?name=${name}`)
      .then((response) => response.json())
      .then(({ items = [] }) => {
        if (items.length === 0) {
          setError('User does not exist')
        } else {
          grantAdminAccess(items[0].id)
        }
      }).catch(e => {
        setError(e)
      })
  }

  return (
    <div className='sm:w-80 lg:w-[500px]'>
      <div className='text-2xs leading-none uppercase text-gray-400 border-b border-gray-800 pb-6'>Admins</div>
      <div className={`flex flex-col sm:flex-row ${error ? 'mt-6 mb-2' : 'mt-6 mb-14'}`}>
        <div className='sm:flex-1'>
          <InputDropdown
            name='name'
            value={name}
            placeholder='Username or email'
            hasDropdownSelection={false}
            handleInputChange={e => handleInputChange(e.target.value)}
            handleKeyDown={(e) => handleKeyDownEvent(e.key)}
            error={error}
          />
        </div>
        <button
          onClick={() => handleAddAdmin()}
          disabled={name.length === 0}
          type='button'
          className='flex items-center cursor-pointer border border-violet-300 px-5 mt-4 text-2xs sm:ml-4 sm:mt-0 rounded-md disabled:pointer-events-none disabled:opacity-30'
        >
          <PlusIcon className='w-3 h-3 mr-1.5' />
          Add Admin
        </button>
      </div>
      {error &&
        <div className='mb-10'>
          <ErrorMessage message={error} />
        </div>}
      <h4 className='text-gray-400 my-3 text-2xs'>These users have full administration privileges</h4>
      {sortedGrants?.map(g => (
        <Grant key={g.id} id={g.id} userID={g.user} grants={sortedGrants} />
      ))}
    </div>
  )
}
