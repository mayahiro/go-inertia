import { InfiniteScroll, useForm, usePage } from '@inertiajs/react'
import type { FormEvent } from 'react'
import type { ExamplePageProps, User } from '../../types'
import Layout from '../Layout'

type PaginatedUsers = {
  data: User[]
}

type UsersIndexProps = {
  users: PaginatedUsers
}

type UserForm = {
  name: string
  email: string
}

export default function UsersIndex({ users }: UsersIndexProps) {
  const { flash } = usePage<ExamplePageProps>().props
  const form = useForm<UserForm>({
    name: '',
    email: '',
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    form.post('/users', {
      preserveState: 'errors',
      reset: ['users'],
      onSuccess: () => form.reset(),
    })
  }

  return (
    <Layout>
      {flash?.success && <div className="flash">{flash.success}</div>}
      <section className="panel">
        <div className="panel-body">
          <InfiniteScroll
            data="users"
            itemsElement="#users-table-body"
            onlyNext
            buffer={160}
            loading={<div className="load-state">Loading more users...</div>}
          >
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Email</th>
                </tr>
              </thead>
              <tbody id="users-table-body">
                {users.data.map(user => (
                  <tr key={user.id}>
                    <td>{user.name}</td>
                    <td>{user.email}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </InfiniteScroll>

          <form className="form" onSubmit={submit}>
            <label className="field">
              <span>Name</span>
              <input
                className="input"
                name="name"
                value={form.data.name}
                onChange={event => form.setData('name', event.target.value)}
              />
              {form.errors.name && <span className="error">{form.errors.name}</span>}
            </label>
            <label className="field">
              <span>Email</span>
              <input
                className="input"
                name="email"
                type="email"
                value={form.data.email}
                onChange={event => form.setData('email', event.target.value)}
              />
              {form.errors.email && <span className="error">{form.errors.email}</span>}
            </label>
            <button className="button" type="submit" disabled={form.processing}>Create</button>
          </form>
        </div>
      </section>
    </Layout>
  )
}
