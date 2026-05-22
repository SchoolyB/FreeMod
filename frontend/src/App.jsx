import { useState, useEffect, useCallback, Component } from 'react'
import {
  ListTrainers,
  LoadTrainer,
  ConnectTrainer,
  ToggleCheat,
  GetTrainerStatus,
  UserTrainerDir,
  ListProcesses,
  AttachProcess,
  ScanValue,
  NarrowValue,
  WriteValue,
  ReadValue,
} from '../wailsjs/go/main/App'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'

// ── Error Boundary ────────────────────────────────────────────────────────────

class ErrorBoundary extends Component {
  constructor(props) { super(props); this.state = { error: null } }
  static getDerivedStateFromError(e) { return { error: e } }
  render() {
    if (this.state.error) return (
      <div className="crash-screen">
        <strong>Render error:</strong> {String(this.state.error)}
        <button onClick={() => this.setState({ error: null })}>Dismiss</button>
      </div>
    )
    return this.props.children
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function Spinner() { return <span className="spinner" /> }

// ── Toggle Switch ─────────────────────────────────────────────────────────────

function Toggle({ checked, onChange, disabled }) {
  return (
    <button
      className={`toggle ${checked ? 'on' : 'off'} ${disabled ? 'disabled' : ''}`}
      onClick={() => !disabled && onChange(!checked)}
      aria-pressed={checked}
    >
      <span className="toggle-knob" />
      <span className="toggle-label">{checked ? 'ON' : 'OFF'}</span>
    </button>
  )
}

// ── Cheat Card ────────────────────────────────────────────────────────────────

function CheatCard({ cheat, disabled, onToggle }) {
  const [inputVal, setInputVal] = useState('')

  const handleToggle = (on) => {
    if (cheat.Input) {
      const num = parseFloat(inputVal)
      if (on && isNaN(num)) return   // don't enable without a value
      onToggle(on, isNaN(num) ? 0 : num)
    } else {
      onToggle(on, 0)
    }
  }

  const handleApply = () => {
    const num = parseFloat(inputVal)
    if (isNaN(num)) return
    // Re-enable with new value (disable first if already on)
    if (cheat.Enabled) onToggle(false, 0)
    setTimeout(() => onToggle(true, num), 50)
  }

  return (
    <div className={`cheat-card ${cheat.Enabled ? 'enabled' : ''}`}>
      <div className="cheat-info">
        <div className="cheat-name">{cheat.Name}</div>
        {cheat.Description && <div className="cheat-desc">{cheat.Description}</div>}
      </div>
      <div className="cheat-controls">
        {cheat.Input && (
          <div className="cheat-input-row">
            <input
              type="number"
              className="cheat-input"
              placeholder="value"
              value={inputVal}
              onChange={e => setInputVal(e.target.value)}
              disabled={disabled}
            />
            {cheat.Enabled && (
              <button className="apply-btn" onClick={handleApply} disabled={disabled || !inputVal}>
                Apply
              </button>
            )}
          </div>
        )}
        <Toggle
          checked={cheat.Enabled}
          disabled={disabled || (cheat.Input && !inputVal && !cheat.Enabled)}
          onChange={handleToggle}
        />
      </div>
    </div>
  )
}

// ── TRAINER TAB ───────────────────────────────────────────────────────────────

function TrainerTab() {
  const [list, setList] = useState([])
  const [loadingList, setLoadingList] = useState(false)
  const [listErr, setListErr] = useState(null)

  const [status, setStatus] = useState(null)   // TrainerStatus from Go
  const [connecting, setConnecting] = useState(false)
  const [toggling, setToggling] = useState(-1) // index of cheat being toggled
  const [actionErr, setActionErr] = useState(null)

  // Load trainer list on mount, and restore any trainer the backend still
  // holds — the Go side stays connected across tab switches, so remounting
  // this tab must not look like a disconnect.
  useEffect(() => {
    setLoadingList(true)
    ListTrainers()
      .then(r => setList(r || []))
      .catch(e => setListErr(String(e)))
      .finally(() => setLoadingList(false))
    GetTrainerStatus()
      .then(s => { if (s && s.Game) setStatus(s) })
      .catch(() => {})
  }, [])

  // Poll status every second to reflect auto-connect/disconnect.
  useEffect(() => {
    if (!status) return
    const id = setInterval(async () => {
      try {
        const s = await GetTrainerStatus()
        if (s && s.Game) setStatus(s)
      } catch (_) {}
    }, 1000)
    return () => clearInterval(id)
  }, [!!status])

  const selectTrainer = useCallback(async (filename) => {
    setActionErr(null)
    try {
      const s = await LoadTrainer(filename)
      setStatus(s)
    } catch (e) {
      setActionErr(String(e))
    }
  }, [])

  const connect = useCallback(async () => {
    setConnecting(true)
    setActionErr(null)
    try {
      const s = await ConnectTrainer()
      setStatus(s)
    } catch (e) {
      setActionErr(String(e))
    } finally {
      setConnecting(false)
    }
  }, [])

  const toggleCheat = useCallback(async (idx, enable, userValue = 0) => {
    setToggling(idx)
    setActionErr(null)
    try {
      const s = await ToggleCheat(idx, enable, userValue)
      setStatus(s)
    } catch (e) {
      setActionErr(String(e))
    } finally {
      setToggling(-1)
    }
  }, [])

  return (
    <div className="trainer-tab">
      {/* Left — game list */}
      <aside className="game-list">
        <div className="panel-header">
          Games
          <button
            className="folder-btn"
            title="Open trainer folder"
            onClick={async () => {
              const dir = await UserTrainerDir()
              BrowserOpenURL('file://' + dir)
            }}
          >📂</button>
        </div>
        {loadingList && <div className="empty"><Spinner /></div>}
        {listErr && <div className="panel-error">{listErr}</div>}
        {list.map(t => (
          <div
            key={t.Filename}
            className={`game-card ${status && status.Game === t.Game ? 'active' : ''}`}
            onClick={() => selectTrainer(t.Filename)}
          >
            <div className="game-card-name">{t.Game}</div>
            <div className="game-card-meta">{t.Exe} · v{t.Version}</div>
          </div>
        ))}
        {!loadingList && list.length === 0 && !listErr && (
          <div className="empty">No trainers found</div>
        )}
      </aside>

      {/* Right — trainer detail */}
      <main className="trainer-detail">
        {!status ? (
          <div className="empty-detail">Select a game to get started</div>
        ) : (
          <>
            {/* Game header */}
            <div className="trainer-header">
              <div className="trainer-title">
                <h1>{status.Game}</h1>
                <span className="trainer-meta">{status.Exe} · v{status.Version}</span>
              </div>
              <div className="trainer-connect">
                {status.Connected ? (
                  <span className="badge running">Running · PID {status.PID}</span>
                ) : (
                  <span className="badge stopped">Not Running</span>
                )}
                <button
                  className="connect-btn"
                  onClick={connect}
                  disabled={connecting}
                >
                  {connecting ? <Spinner /> : status.Connected ? 'Reconnect' : 'Connect'}
                </button>
              </div>
            </div>

            {actionErr && <div className="action-error">{actionErr}</div>}

            {/* Cheat list */}
            <div className="cheat-list">
              {(status.Cheats || []).map((cheat, idx) => (
                <CheatCard
                  key={idx}
                  cheat={cheat}
                  disabled={!status.Connected || toggling === idx}
                  onToggle={(on, userValue) => toggleCheat(idx, on, userValue)}
                />
              ))}
            </div>
          </>
        )}
      </main>
    </div>
  )
}

// ── DEV MODE TAB ──────────────────────────────────────────────────────────────

function DevModeTab() {
  const [attachedPID, setAttachedPID] = useState(0)
  const [attachedName, setAttachedName] = useState('')
  const [procs, setProcs] = useState([])
  const [filter, setFilter] = useState('')
  const [loadingProcs, setLoadingProcs] = useState(false)
  const [procErr, setProcErr] = useState(null)

  const [addresses, setAddresses] = useState([])
  const [scanVal, setScanVal] = useState('')
  const [scanning, setScanning] = useState(false)

  const [writeAddr, setWriteAddr] = useState('')
  const [writeVal, setWriteVal] = useState('')
  const [readResult, setReadResult] = useState(null)
  const [writeMsg, setWriteMsg] = useState(null)

  const [status, setStatus] = useState(null)

  const showStatus = (text, isError = false) => setStatus({ text, isError })

  const refreshProcs = useCallback(async () => {
    setLoadingProcs(true)
    setProcErr(null)
    try {
      const list = await ListProcesses()
      setProcs(list || [])
    } catch (e) {
      setProcErr(String(e))
    } finally {
      setLoadingProcs(false)
    }
  }, [])

  const attach = useCallback(async (pid, name) => {
    try {
      await AttachProcess(pid)
      setAttachedPID(pid)
      setAttachedName(name)
      setAddresses([])
      showStatus(`Attached to ${name} (PID ${pid})`)
    } catch (e) {
      showStatus(String(e), true)
    }
  }, [])

  const doScan = useCallback(async (narrow) => {
    const num = parseInt(scanVal, 10)
    if (isNaN(num)) return
    setScanning(true)
    try {
      const fn = narrow ? NarrowValue : ScanValue
      const r = await fn(num)
      setAddresses(r.Addresses || [])
      showStatus(`${narrow ? 'Narrowed' : 'Scanned'} — ${r.Count} match${r.Count === 1 ? '' : 'es'}`)
    } catch (e) {
      showStatus(String(e), true)
    } finally {
      setScanning(false)
    }
  }, [scanVal])

  const doRead = useCallback(async () => {
    try {
      const v = await ReadValue(writeAddr)
      setReadResult(v)
    } catch (e) {
      setWriteMsg({ text: String(e), err: true })
    }
  }, [writeAddr])

  const doWrite = useCallback(async () => {
    const num = parseInt(writeVal, 10)
    if (!writeAddr || isNaN(num)) return
    try {
      await WriteValue(writeAddr, num)
      setWriteMsg({ text: `Wrote ${num} to ${writeAddr}`, err: false })
      setReadResult(num)
    } catch (e) {
      setWriteMsg({ text: String(e), err: true })
    }
  }, [writeAddr, writeVal])

  const filtered = procs.filter(p =>
    (p.Name || '').toLowerCase().includes(filter.toLowerCase()) ||
    String(p.PID).includes(filter)
  )

  return (
    <div className="dev-tab">
      {/* Status bar */}
      {status && (
        <div className={`dev-status ${status.isError ? 'error' : 'ok'}`}>
          {status.text}
        </div>
      )}

      <div className="dev-layout">
        {/* Process panel */}
        <section className="dev-card">
          <div className="dev-card-header">
            <span>Process</span>
            {attachedPID > 0 && <span className="badge running">PID {attachedPID}</span>}
            <button onClick={refreshProcs} disabled={loadingProcs}>
              {loadingProcs ? <Spinner /> : 'Refresh'}
            </button>
          </div>
          <input
            className="dev-filter"
            placeholder="Filter…"
            value={filter}
            onChange={e => setFilter(e.target.value)}
          />
          {procErr && <div className="panel-error">{procErr}</div>}
          <div className="dev-proc-list">
            {filtered.length === 0 && !procErr && (
              <div className="empty">Click Refresh to list processes</div>
            )}
            {filtered.map(p => (
              <div
                key={p.PID}
                className={`dev-proc-row ${p.PID === attachedPID ? 'selected' : ''}`}
                onClick={() => attach(p.PID, p.Name)}
              >
                <span className="proc-pid">{p.PID}</span>
                <span className="proc-name">{p.Name}</span>
              </div>
            ))}
          </div>
        </section>

        <div className="dev-right">
          {/* Scanner */}
          <section className="dev-card">
            <div className="dev-card-header"><span>Scanner</span></div>
            <div className="dev-row">
              <input
                type="number"
                placeholder="Value (int32)"
                value={scanVal}
                onChange={e => setScanVal(e.target.value)}
                disabled={!attachedPID || scanning}
              />
              <button onClick={() => doScan(false)} disabled={!attachedPID || scanning || !scanVal}>
                {scanning ? <Spinner /> : 'Scan'}
              </button>
              <button onClick={() => doScan(true)} disabled={!attachedPID || scanning || !scanVal || !addresses.length} className="secondary">
                Narrow
              </button>
            </div>
            <div className="dev-results-hdr">
              {addresses.length > 0
                ? `${addresses.length} result${addresses.length === 1 ? '' : 's'}${addresses.length > 200 ? ' (showing 200)' : ''}`
                : 'No results'}
              {addresses.length === 1 && <span className="found-badge">Found!</span>}
            </div>
            <div className="dev-addr-list">
              {(addresses.length > 200 ? addresses.slice(0, 200) : addresses).map(addr => (
                <div key={addr} className="dev-addr-row" onClick={() => setWriteAddr(addr)}>
                  {addr}
                </div>
              ))}
            </div>
          </section>

          {/* Write */}
          <section className="dev-card">
            <div className="dev-card-header"><span>Write</span></div>
            <div className="dev-row">
              <input
                placeholder="Address (0x…)"
                value={writeAddr}
                onChange={e => { setWriteAddr(e.target.value); setReadResult(null) }}
                disabled={!attachedPID}
              />
              <button onClick={doRead} disabled={!attachedPID || !writeAddr} className="secondary">
                Read
              </button>
            </div>
            {readResult !== null && (
              <div className="dev-current-val">Current: <strong>{readResult}</strong></div>
            )}
            <div className="dev-row">
              <input
                type="number"
                placeholder="New value (int32)"
                value={writeVal}
                onChange={e => setWriteVal(e.target.value)}
                disabled={!attachedPID}
              />
              <button onClick={doWrite} disabled={!attachedPID || !writeAddr || !writeVal} className="danger">
                Write
              </button>
            </div>
            {writeMsg && (
              <div className={`dev-write-msg ${writeMsg.err ? 'error' : 'ok'}`}>
                {writeMsg.text}
              </div>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}

// ── Root App ──────────────────────────────────────────────────────────────────

export default function App() {
  const [tab, setTab] = useState('trainers')

  return (
    <ErrorBoundary>
      <div className="app">
        <header className="app-header">
          <span className="logo">FreeMod</span>
          <nav className="tab-bar">
            <button
              className={`tab-btn ${tab === 'trainers' ? 'active' : ''}`}
              onClick={() => setTab('trainers')}
            >
              Trainers
            </button>
            <button
              className={`tab-btn ${tab === 'dev' ? 'active' : ''}`}
              onClick={() => setTab('dev')}
            >
              Dev Mode
            </button>
          </nav>
        </header>

        <div className="app-body">
          {tab === 'trainers' ? <TrainerTab /> : <DevModeTab />}
        </div>
      </div>
    </ErrorBoundary>
  )
}
