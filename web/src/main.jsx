import { useEffect, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import {
  ArrowDownLeft,
  ArrowUpRight,
  Check,
  ChevronDown,
  CircleDollarSign,
  ClipboardList,
  LogOut,
  Menu,
  Plus,
  ReceiptText,
  Send,
  Settings2,
  Sparkles,
  Users,
  X,
} from 'lucide-react'
import './styles.css'

const API_BASE = import.meta.env.VITE_API_BASE || ''
const initialForm = { description: '', amount_cents: '', currency: 'INR', paid_by_user_id: '', expense_date: new Date().toISOString().slice(0, 10), splits: [] }

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
  })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error || 'Something went wrong')
  }
  return response.status === 204 ? null : response.json()
}

function initials(name = '') {
  return name.split(' ').map((part) => part[0]).join('').slice(0, 2).toUpperCase() || '?'
}

function money(cents, currency = 'INR') {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency, minimumFractionDigits: 2, maximumFractionDigits: 2 }).format((cents || 0) / 100)
}

function rupeesToCents(value) {
  const amount = Number(value)
  return Number.isFinite(amount) ? Math.round(amount * 100) : 0
}

function App() {
  const [token, setToken] = useState(localStorage.getItem('expense_token'))
  const [user, setUser] = useState(null)
  const [groups, setGroups] = useState([])
  const [selectedGroup, setSelectedGroup] = useState(null)
  const [members, setMembers] = useState([])
  const [expenses, setExpenses] = useState([])
  const [balances, setBalances] = useState([])
  const [suggestions, setSuggestions] = useState([])
  const [modal, setModal] = useState(null)
  const [profileOpen, setProfileOpen] = useState(false)
  const [preferencesOpen, setPreferencesOpen] = useState(false)
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const groupLoadSequence = useRef(0)

  const auth = token ? { Authorization: `Bearer ${token}` } : {}
  const api = (path, options = {}) => request(path, { ...options, headers: { ...auth, ...(options.headers || {}) } })

  async function loadUser() {
    const me = await api('/v1/me')
    const availableGroups = await api('/v1/groups')
    setUser(me)
    setGroups(availableGroups)
    setSelectedGroup((current) => availableGroups.find((group) => group.id === current?.id) || availableGroups[0] || null)
  }

  async function loadGroup(group) {
    const sequence = ++groupLoadSequence.current
    if (!group) {
      setMembers([])
      setExpenses([])
      setBalances([])
      setSuggestions([])
      return
    }
    setLoading(true)
    setError('')
    try {
      const [groupMembers, groupExpenses, groupBalances, groupSuggestions] = await Promise.all([
        api(`/v1/groups/${group.id}/members`),
        api(`/v1/groups/${group.id}/expenses?limit=50`),
        api(`/v1/groups/${group.id}/balances`),
        api(`/v1/groups/${group.id}/settlement-suggestions`),
      ])
      if (sequence !== groupLoadSequence.current) return
      setMembers(groupMembers)
      setExpenses(groupExpenses)
      setBalances(groupBalances)
      setSuggestions(groupSuggestions)
    } catch (err) {
      if (sequence === groupLoadSequence.current) setError(err.message)
    } finally {
      if (sequence === groupLoadSequence.current) setLoading(false)
    }
  }

  useEffect(() => { if (token) loadUser().catch((err) => { localStorage.removeItem('expense_token'); setToken(null); setError(err.message) }) }, [token])
  useEffect(() => { loadGroup(selectedGroup) }, [selectedGroup])

  async function refresh() { await loadUser(); await loadGroup(selectedGroup) }

  async function deleteGroup() {
    if (!selectedGroup || !window.confirm(`Delete ${selectedGroup.name}? Groups with financial records must be cleared first.`)) return
    try {
      await api(`/v1/groups/${selectedGroup.id}`, { method: 'DELETE' })
      await loadUser()
    } catch (err) { setError(err.message) }
  }

  async function deleteExpense(expense) {
    if (!window.confirm(`Delete “${expense.description}”?`)) return
    try {
      await api(`/v1/groups/${selectedGroup.id}/expenses/${expense.id}`, { method: 'DELETE' })
      await loadGroup(selectedGroup)
    } catch (err) { setError(err.message) }
  }

  function signOut() { api('/v1/auth/session', { method: 'DELETE' }).catch(() => {}); localStorage.removeItem('expense_token'); setToken(null); setUser(null) }

  if (!token) return <AuthScreen onSignedIn={(nextToken) => { localStorage.setItem('expense_token', nextToken); setToken(nextToken) }} />

  return <>
    <div className="app-shell">
      <aside className={`sidebar ${mobileNavOpen ? 'mobile-open' : ''}`}>
        <div className="brand"><span className="brand-mark"><CircleDollarSign size={21} /></span><span>ledger<span className="brand-dot">.</span></span></div>
        <div className="sidebar-label">Your spaces</div>
        <div className="group-list">
          {groups.map((group) => <button className={`group-item ${selectedGroup?.id === group.id ? 'active' : ''}`} key={group.id} onClick={() => setSelectedGroup(group)}><span className="group-icon">{initials(group.name)}</span><span>{group.name}</span>{selectedGroup?.id === group.id && <ChevronDown size={15} />}</button>)}
          <button className="new-space" onClick={() => setModal('group')}><Plus size={16} /> New space</button>
        </div>
        <div className="sidebar-bottom"><button className="quiet-button" onClick={() => { setPreferencesOpen(true); setMobileNavOpen(false) }}><Settings2 size={17} /> Preferences</button><button className="quiet-button" onClick={signOut}><LogOut size={17} /> Sign out</button></div>
      </aside>

      <main className="main-content">
        <header className="topbar"><button className="mobile-menu" aria-label="Open menu" onClick={() => setMobileNavOpen(!mobileNavOpen)}><Menu size={22} /></button><div className="breadcrumb">Spaces <span>/</span> <strong>{selectedGroup?.name || 'No space yet'}</strong></div><div className="profile-wrap"><button className="profile" onClick={() => setProfileOpen(!profileOpen)}><span className="avatar coral">{initials(user?.display_name)}</span><span className="profile-name">{user?.display_name}</span><ChevronDown size={15} /></button>{profileOpen && <div className="profile-menu"><strong>{user?.display_name}</strong><span>{user?.email}</span><button onClick={() => { setPreferencesOpen(true); setProfileOpen(false) }}><Settings2 size={15} /> Preferences</button><button onClick={signOut}><LogOut size={15} /> Sign out</button></div>}</div></header>
        {error && <div className="error-banner"><span>{error}</span><button onClick={() => setError('')}><X size={16} /></button></div>}
        {!selectedGroup ? <EmptyState onCreate={() => setModal('group')} /> : <>
          <section className="page-heading"><div><div className="eyebrow">Overview <span className="live-dot" /> Live</div><h1>{selectedGroup.name}</h1><p>Every shared spend, in one calm place.</p></div><div className="heading-actions"><button className="danger-button" onClick={deleteGroup}><X size={16} /> Delete group</button><button className="outline-button" onClick={() => setModal('member')}><Users size={17} /> Add member</button><button className="primary-button" onClick={() => setModal('expense')}><Plus size={18} /> Add expense</button></div></section>
          <section className="balance-strip"><div className="balance-intro"><span className="balance-icon"><Sparkles size={18} /></span><div><span className="muted-label">Group balance</span><strong>Shared picture</strong></div></div>{balances.flatMap((balance) => balance.members.map((member) => <div className={`member-balance ${member.balance_cents >= 0 ? 'positive' : 'negative'}`} key={`${balance.currency}-${member.user.id}`}><span className="avatar small">{initials(member.user.display_name)}</span><div><span>{member.user.display_name} <small>{balance.currency}</small></span><strong>{member.balance_cents === 0 ? 'Settled up' : `${member.balance_cents > 0 ? '+' : '-'}${money(Math.abs(member.balance_cents), balance.currency)}`}</strong></div></div>))}</section>
          <div className="content-grid"><section className="ledger-panel"><div className="section-heading"><div><span className="eyebrow">Recent activity</span><h2>The ledger</h2></div><span className="count-pill">{expenses.length} entries</span></div>{loading ? <div className="loading">Updating the ledger...</div> : expenses.length === 0 ? <div className="empty-panel"><ReceiptText size={28} /><strong>No expenses yet</strong><span>Your first shared spend belongs here.</span></div> : <div className="expense-list">{expenses.map((expense) => <ExpenseRow expense={expense} key={expense.id} onEdit={() => setModal({ type: 'edit-expense', expense })} onDelete={() => deleteExpense(expense)} />)}</div>}</section>
            <aside className="right-column"><section className="suggestions-panel"><div className="section-heading compact"><div><span className="eyebrow">Suggested next</span><h2>Settle up</h2></div><Sparkles size={18} className="spark-icon" /></div>{suggestions.length === 0 ? <div className="settled-state"><span className="check-circle"><Check size={18} /></span><strong>All clear</strong><span>Everyone is settled for now.</span></div> : <div className="suggestion-list">{suggestions.slice(0, 4).map((suggestion, index) => <div className="suggestion" key={`${suggestion.from.id}-${suggestion.to.id}-${index}`}><div className="transfer-people"><span className="avatar tiny">{initials(suggestion.from.display_name)}</span><ArrowDownLeft size={14} /><span className="avatar tiny mint">{initials(suggestion.to.display_name)}</span></div><div><strong>{suggestion.from.display_name} <span>pays</span></strong><small>{suggestion.to.display_name}</small></div><b>{money(suggestion.amount_cents, suggestion.currency)}</b></div>)}</div>}</section><section className="members-panel"><div className="section-heading compact"><div><span className="eyebrow">The circle</span><h2>Members <span className="member-count">{members.length}</span></h2></div><button className="icon-button" onClick={() => setModal('member')} aria-label="Add member"><Plus size={18} /></button></div><div className="member-roster">{members.map((member, index) => <div className="roster-row" key={member.id}><span className={`avatar ${index % 3 === 1 ? 'mint' : index % 3 === 2 ? 'yellow' : ''}`}>{initials(member.display_name)}</span><div><strong>{member.display_name}</strong><small>{member.id === user?.id ? 'You' : member.email}</small></div><span className="roster-status" /></div>)}</div></section></aside></div>
        </>}
        {selectedGroup && suggestions[0] && <button className="settle-floating" onClick={() => setModal({ type: 'settlement', suggestion: suggestions[0] })}><Send size={15} /> Record suggested payment</button>}
      </main>
    </div>
    {modal === 'group' && <GroupModal onClose={() => setModal(null)} onCreated={async (group) => { setModal(null); await loadUser(); setSelectedGroup(group) }} api={api} />}
    {modal === 'member' && <MemberModal members={members} onClose={() => setModal(null)} onCreated={async () => { setModal(null); await loadGroup(selectedGroup) }} api={api} group={selectedGroup} />}
    {modal === 'expense' && <ExpenseModal members={members} user={user} onClose={() => setModal(null)} onCreated={async () => { setModal(null); await loadGroup(selectedGroup) }} api={api} group={selectedGroup} />}
    {modal?.type === 'edit-expense' && <ExpenseModal expense={modal.expense} members={members} user={user} onClose={() => setModal(null)} onCreated={async () => { setModal(null); await loadGroup(selectedGroup) }} api={api} group={selectedGroup} />}
    {modal?.type === 'settlement' && <SettlementModal suggestion={modal.suggestion} onClose={() => setModal(null)} onCreated={async () => { setModal(null); await loadGroup(selectedGroup) }} api={api} group={selectedGroup} />}
    {preferencesOpen && <PreferencesModal user={user} api={api} onSaved={async () => { setPreferencesOpen(false); await loadUser() }} onClose={() => setPreferencesOpen(false)} />}
  </>
}

function ExpenseRow({ expense, onEdit, onDelete }) { return <article className="expense-row"><div className="date-block"><strong>{new Date(`${expense.expense_date}T12:00:00`).toLocaleDateString('en-US', { day: '2-digit' })}</strong><span>{new Date(`${expense.expense_date}T12:00:00`).toLocaleDateString('en-US', { month: 'short' }).toUpperCase()}</span></div><div className="expense-icon"><ReceiptText size={18} /></div><div className="expense-main"><strong>{expense.description}</strong><span>Paid by {expense.paid_by.display_name} <i>·</i> split {expense.splits.length} ways</span></div><div className="expense-amount"><strong>{money(expense.amount_cents, expense.currency)}</strong><span>{expense.currency}</span></div><div className="expense-actions"><button onClick={onEdit} aria-label={`Edit ${expense.description}`}>Edit</button><button onClick={onDelete} aria-label={`Delete ${expense.description}`}>Delete</button></div></article> }

function AuthScreen({ onSignedIn }) { const [mode, setMode] = useState('login'); const [form, setForm] = useState({ email: '', password: '', display_name: '' }); const [error, setError] = useState(''); const [working, setWorking] = useState(false); async function submit(event) { event.preventDefault(); const normalized = { email: form.email.trim(), password: form.password }; const payload = mode === 'register' ? { ...normalized, display_name: form.display_name.trim() } : normalized; if (!payload.email || !payload.password || (mode === 'register' && !payload.display_name)) { setError('Email and password are required.'); return } setWorking(true); setError(''); try { const result = await request(`/v1/auth/${mode === 'login' ? 'login' : 'register'}`, { method: 'POST', body: JSON.stringify(payload) }); onSignedIn(result.session_token) } catch (err) { setError(err.message) } finally { setWorking(false) } } return <div className="auth-page"><div className="auth-art"><div className="brand light"><span className="brand-mark"><CircleDollarSign size={21} /></span><span>ledger<span className="brand-dot">.</span></span></div><div className="art-copy"><span className="eyebrow light-text">A better way to split</span><h1>Keep the good times.<br /><em>Lose the math.</em></h1><p>A shared ledger for the dinners, trips, and little moments that are better together.</p></div><div className="art-footer"><span>01</span><div className="art-line" /><span>MAKE IT EVEN</span></div></div><div className="auth-card"><div className="mobile-brand"><div className="brand"><span className="brand-mark"><CircleDollarSign size={21} /></span><span>ledger<span className="brand-dot">.</span></span></div></div><div className="auth-heading"><span className="eyebrow">Welcome back</span><h2>{mode === 'login' ? 'Your shared life awaits.' : 'Start a shared space.'}</h2><p>{mode === 'login' ? 'Sign in to pick up where you left off.' : 'Create your account and invite your people.'}</p></div><form onSubmit={submit}>{mode === 'register' && <label>Name<input required value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} placeholder="What should we call you?" /></label>}<label>Email<input required type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder="you@example.com" /></label><label>Password<input required minLength="12" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="12 characters minimum" /></label>{error && <div className="form-error">{error}</div>}<button className="primary-button wide" disabled={working}>{working ? 'Opening...' : mode === 'login' ? 'Enter ledger' : 'Create account'} <ArrowUpRight size={17} /></button></form><button className="switch-auth" onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError('') }}>{mode === 'login' ? 'New here? Create an account' : 'Already have an account? Sign in'}</button></div></div> }

function Modal({ title, eyebrow, children, onClose }) { return <div className="modal-backdrop" onMouseDown={(event) => event.target === event.currentTarget && onClose()}><div className="modal"><button className="modal-close" onClick={onClose} aria-label="Close"><X size={19} /></button><span className="eyebrow">{eyebrow}</span><h2>{title}</h2>{children}</div></div> }
function GroupModal({ api, onClose, onCreated }) { const [name, setName] = useState(''); const [error, setError] = useState(''); async function submit(e) { e.preventDefault(); try { const group = await api('/v1/groups', { method: 'POST', body: JSON.stringify({ name }) }); onCreated(group) } catch (err) { setError(err.message) } } return <Modal title="Create a new space" eyebrow="New shared ledger" onClose={onClose}><form onSubmit={submit}><label>Space name<input autoFocus required value={name} onChange={(e) => setName(e.target.value)} placeholder="Weekend in Goa" /></label>{error && <div className="form-error">{error}</div>}<button className="primary-button wide">Create space <ArrowUpRight size={17} /></button></form></Modal> }
function MemberModal({ api, group, onClose, onCreated }) { const [email, setEmail] = useState(''); const [error, setError] = useState(''); async function submit(e) { e.preventDefault(); try { await api(`/v1/groups/${group.id}/members`, { method: 'POST', body: JSON.stringify({ email }) }); onCreated() } catch (err) { setError(err.message) } } return <Modal title="Bring someone in" eyebrow={`Invite to ${group.name}`} onClose={onClose}><form onSubmit={submit}><label>Email address<input autoFocus required type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="friend@example.com" /></label><p className="modal-note">They need an account before they can join this space.</p>{error && <div className="form-error">{error}</div>}<button className="primary-button wide">Add to circle <Users size={17} /></button></form></Modal> }
function ExpenseModal({ api, group, members, user, expense, onClose, onCreated }) {
  const buildEqualSplits = (amountValue, memberList) => {
    const amount = Number(amountValue || 0)
    const totalCents = Math.round(amount * 100)
    if (!memberList.length || !totalCents) {
      return memberList.map((member) => ({ user_id: member.id, amount_cents: '' }))
    }
    const base = Math.floor(totalCents / memberList.length)
    const remainder = totalCents - (base * memberList.length)
    return memberList.map((member, index) => ({
      user_id: member.id,
      amount_cents: ((index === 0 ? base + remainder : base) / 100).toFixed(2),
    }))
  }

  const buildPercentageSplits = (amountValue, memberList, percentageMap) => {
    const amount = Number(amountValue || 0)
    const totalCents = Math.round(amount * 100)
    if (!memberList.length || !totalCents) {
      return memberList.map((member) => ({ user_id: member.id, amount_cents: '' }))
    }

    const totalPercent = memberList.reduce((sum, member) => sum + (Number(percentageMap[member.id]) || 0), 0)
    if (!totalPercent) {
      return buildEqualSplits(amountValue, memberList)
    }

    let remainingCents = totalCents
    const parts = memberList.map((member, index) => {
      const share = Number(percentageMap[member.id]) || 0
      const raw = (totalCents * share) / totalPercent
      const rounded = index === memberList.length - 1 ? remainingCents : Math.round(raw)
      remainingCents -= rounded
      return {
        user_id: member.id,
        amount_cents: (rounded / 100).toFixed(2),
      }
    })

    return parts
  }

  const [form, setForm] = useState(() => ({
    ...initialForm,
    description: expense?.description || '',
    amount_cents: expense ? expense.amount_cents / 100 : '',
    currency: expense?.currency || 'INR',
    expense_date: expense?.expense_date || initialForm.expense_date,
    paid_by_user_id: expense?.paid_by_user_id || expense?.paid_by?.id || user?.id,
    splits: expense ? expense.splits.map((split) => ({ user_id: split.user.id, amount_cents: split.amount_cents / 100 })) : members.map((member) => ({ user_id: member.id, amount_cents: '' })),
  }))
  const [splitMode, setSplitMode] = useState(expense ? 'custom' : 'equal')
  const [percentageInputs, setPercentageInputs] = useState(() => Object.fromEntries(members.map((member) => [member.id, members.length ? Number((100 / members.length).toFixed(2)) : 0])))
  const [error, setError] = useState('')

  function setField(field, value) {
    const next = { ...form, [field]: value }
    if (field === 'amount_cents' && splitMode !== 'custom') {
      next.splits = splitMode === 'equal'
        ? buildEqualSplits(value, members)
        : buildPercentageSplits(value, members, percentageInputs)
      setForm(next)
      return
    }
    setForm(next)
  }

  function setSplit(id, value) {
    setForm({
      ...form,
      splits: form.splits.map((split) => split.user_id === id ? { ...split, amount_cents: value } : split),
    })
  }

  function updatePercentage(id, value) {
    const sanitized = Math.max(0, Number(value) || 0)
    const nextPercentages = { ...percentageInputs, [id]: sanitized }
    setPercentageInputs(nextPercentages)
    if (splitMode === 'percentage') {
      setForm({
        ...form,
        splits: buildPercentageSplits(form.amount_cents, members, nextPercentages),
      })
    }
  }

  function applySplitMode(mode) {
    if (!members.length) {
      setSplitMode(mode)
      return
    }

    if (mode === 'equal') {
      setForm({ ...form, splits: buildEqualSplits(form.amount_cents, members) })
    } else if (mode === 'percentage') {
      setForm({ ...form, splits: buildPercentageSplits(form.amount_cents, members, percentageInputs) })
    } else {
      setForm({
        ...form,
        splits: form.splits.length ? form.splits : members.map((member) => ({ user_id: member.id, amount_cents: '' })),
      })
    }

    setSplitMode(mode)
  }

  async function submit(e) {
    e.preventDefault();
    const amountCents = rupeesToCents(form.amount_cents)
    const splits = form.splits.filter((split) => Number(split.amount_cents) > 0).map((split) => ({ ...split, amount_cents: rupeesToCents(split.amount_cents) }))
    if (!amountCents || splits.reduce((total, split) => total + split.amount_cents, 0) !== amountCents) {
      setError('Split amounts must add up to the total amount.')
      return
    }
    try {
      await api(`/v1/groups/${group.id}/expenses${expense ? `/${expense.id}` : ''}`, { method: expense ? 'PATCH' : 'POST', body: JSON.stringify({ ...form, amount_cents: amountCents, splits }) })
      onCreated()
    } catch (err) {
      setError(err.message)
    }
  }

  return <Modal title={expense ? 'Edit expense' : 'Add an expense'} eyebrow={expense ? 'Update ledger entry' : 'New ledger entry'} onClose={onClose}><form onSubmit={submit} className="expense-form"><label>What was it for?<input autoFocus required value={form.description} onChange={(e) => setField('description', e.target.value)} placeholder="Dinner at Little Italy" /></label><div className="form-row"><label>Amount (₹)<input required type="number" min="0.01" step="0.01" value={form.amount_cents} onChange={(e) => setField('amount_cents', e.target.value)} placeholder="1200" /></label><label>Currency<input maxLength="3" required value={form.currency} onChange={(e) => setField('currency', e.target.value.toUpperCase())} /></label></div><div className="split-section"><div className="split-header"><h3>Split</h3><select value={splitMode} onChange={(e) => applySplitMode(e.target.value)}><option value="equal">Equal</option><option value="percentage">Percentage</option><option value="custom">Custom</option></select></div>{splitMode === 'percentage' && <div className="split-percentages">{members.map((member) => <label key={member.id}>{member.display_name}<input type="number" min="0" max="100" step="1" value={percentageInputs[member.id] || 0} onChange={(e) => updatePercentage(member.id, e.target.value)} /></label>)}</div>}{splitMode === 'custom' ? <div className="split-list">{form.splits.map((split) => { const member = members.find((item) => item.id === split.user_id); if (!member) return null; return <div className="split-row" key={split.user_id}><span>{member.display_name}</span><input type="number" min="0" step="0.01" value={split.amount_cents} onChange={(e) => setSplit(split.user_id, e.target.value)} /></div> })}</div> : <div className="split-list">{form.splits.map((split) => { const member = members.find((item) => item.id === split.user_id); if (!member) return null; return <div className="split-row" key={split.user_id}><span>{member.display_name}</span><input type="number" min="0" step="0.01" value={split.amount_cents} readOnly /></div> })}</div>}</div><label>Paid by<select value={form.paid_by_user_id} onChange={(e) => setField('paid_by_user_id', e.target.value)}>{members.map((member) => <option key={member.id} value={member.id}>{member.display_name}</option>)}</select></label><label>Expense date<input type="date" value={form.expense_date} onChange={(e) => setField('expense_date', e.target.value)} /></label>{error && <div className="form-error">{error}</div>}<button className="primary-button wide">{expense ? 'Save changes' : 'Add expense'} <ArrowUpRight size={17} /></button></form></Modal> }

function PreferencesModal({ user, api, onSaved, onClose }) { const [upiID, setUpiID] = useState(user?.upi_id || ''); const [error, setError] = useState(''); async function save(event) { event.preventDefault(); try { await api('/v1/me', { method: 'PATCH', body: JSON.stringify({ upi_id: upiID.trim() }) }); onSaved() } catch (err) { setError(err.message) } } return <Modal title="Preferences" eyebrow="Your account" onClose={onClose}><form onSubmit={save}><div className="preference-list"><div><span>Signed in as</span><strong>{user?.email}</strong></div><label>UPI ID<input value={upiID} onChange={(event) => setUpiID(event.target.value)} placeholder="yourname@upi" /></label><div><span>Default currency</span><strong>INR</strong></div></div>{error && <div className="form-error">{error}</div>}<button className="primary-button wide">Save UPI ID <Check size={17} /></button></form></Modal> }
function SettlementModal({ api, group, suggestion, onClose, onCreated }) { const [note, setNote] = useState(''); const [error, setError] = useState(''); async function submit(event) { event.preventDefault(); try { await api(`/v1/groups/${group.id}/settlements`, { method: 'POST', body: JSON.stringify({ received_by_user_id: suggestion.to.id, amount_cents: suggestion.amount_cents, currency: suggestion.currency, note }) }); onCreated() } catch (err) { setError(err.message) } } return <Modal title="Record payment" eyebrow="Settlement / UPI note" onClose={onClose}><div className="settlement-summary"><span className="avatar small">{initials(suggestion.from.display_name)}</span><strong>{suggestion.from.display_name}</strong><ArrowUpRight size={16} /><span className="avatar small mint">{initials(suggestion.to.display_name)}</span><strong>{suggestion.to.display_name}</strong><b>{money(suggestion.amount_cents, suggestion.currency)}</b></div><div className="upi-destination"><span>Pay to UPI ID</span><strong>{suggestion.to.upi_id || 'Receiver has not added a UPI ID yet'}</strong></div><form onSubmit={submit}><label>Payment note (optional)<input autoFocus value={note} onChange={(event) => setNote(event.target.value)} placeholder="UPI payment, cash, bank transfer" /></label>{error && <div className="form-error">{error}</div>}<button className="primary-button wide"><Check size={17} /> Mark as paid</button></form></Modal> }

function EmptyState({ onCreate }) { return <div className="empty-workspace"><div className="empty-orbit"><ClipboardList size={35} /></div><span className="eyebrow">Your first shared ledger</span><h1>Make room for<br /><em>good memories.</em></h1><p>Create a space for a trip, a home, or simply keeping life even.</p><button className="primary-button" onClick={onCreate}><Plus size={18} /> Create a space</button></div> }

export default App

createRoot(document.getElementById('root')).render(<App />)