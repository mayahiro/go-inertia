import { useForm, usePage } from '@inertiajs/react'
import type { FormEvent } from 'react'
import type { ExamplePageProps, User } from '../../types'
import Layout from '../Layout'

type UsersIndexProps = {
  users: User[]
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
      onSuccess: () => form.reset(),
    })
  }

  return (
    <Layout>
      {flash?.success && <div className="flash">{flash.success}</div>}
      <section className="panel">
        <div className="panel-body">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.id}>
                  <td>{user.name}</td>
                  <td>{user.email}</td>
                </tr>
              ))}
            </tbody>
          </table>

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
