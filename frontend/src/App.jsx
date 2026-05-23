import { useState, useEffect, useCallback, Component } from 'react'
import {
  ListTrainers,
  LoadTrainer,
  ConnectTrainer,
  ToggleCheat,
  GetTrainerStatus,
  UserTrainerDir,
  TrainerImage,
  DisconnectTrainer,
  KillApp,
  ListProcesses,
  AttachProcess,
  GetModuleBase,
  ScanValue,
  NarrowValue,
  ScanByMode,
  WriteValue,
  ReadValue,
  ReadAllTypes,
  TestFreeze,
  StopTestFreeze,
  GetSettings,
  SaveSettings,
  OpenFolder,
  PickTrainerDir,
  SetFullscreen,
  ScanValueFloat,
  NarrowValueFloat,
  WalkPointerChain,
  ScanAllTypes,
  NarrowAllTypes,
  ScanByModeAll,
} from '../wailsjs/go/main/App'

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
  const behavior = cheat.Behavior || 'freeze'
  const isThreshold = behavior === 'threshold'
  const isClamp = behavior === 'clamp_min' || behavior === 'clamp_max'

  const [valueInput, setValueInput] = useState(cheat.Input ? String(cheat.Value || '') : '')
  const [triggerInput, setTriggerInput] = useState(cheat.Input && isThreshold ? String(cheat.Trigger || '') : '')

  const handleToggle = (on) => {
    if (!on) { onToggle(false, 0, 0); return }
    if (cheat.Input) {
      const val = parseFloat(valueInput)
      if (isNaN(val)) return
      const trig = isThreshold ? parseFloat(triggerInput) : 0
      if (isThreshold && isNaN(trig)) return
      onToggle(true, val, trig)
    } else {
      onToggle(true, 0, 0)
    }
  }

  const handleApply = () => {
    const val = parseFloat(valueInput)
    if (isNaN(val)) return
    const trig = isThreshold ? parseFloat(triggerInput) : 0
    if (cheat.Enabled) onToggle(false, 0, 0)
    setTimeout(() => onToggle(true, val, trig), 50)
  }

  const canEnable = !cheat.Input ||
    (valueInput !== '' && !isNaN(parseFloat(valueInput)) &&
      (!isThreshold || (triggerInput !== '' && !isNaN(parseFloat(triggerInput)))))

  return (
    <div className={`cheat-card ${cheat.Enabled ? 'enabled' : ''} behavior-${behavior}`}>
      <div className="cheat-info">
        <div className="cheat-name">
          {cheat.Name}
          {behavior !== 'freeze' && (
            <span className={`behavior-badge behavior-badge-${behavior}`}>
              {isThreshold ? 'threshold' : isClamp ? behavior === 'clamp_min' ? 'floor' : 'ceiling' : ''}
            </span>
          )}
        </div>
        {cheat.Description && <div className="cheat-desc">{cheat.Description}</div>}
      </div>
      <div className="cheat-controls">
        {cheat.Input && isThreshold && (
          <div className="cheat-threshold-row">
            <span className="threshold-label">if &lt;</span>
            <input
              type="number"
              className="cheat-input cheat-input-sm"
              placeholder={String(cheat.Trigger)}
              value={triggerInput}
              onChange={e => setTriggerInput(e.target.value)}
              disabled={disabled || cheat.Enabled}
            />
            <span className="threshold-label">→</span>
            <input
              type="number"
              className="cheat-input cheat-input-sm"
              placeholder={String(cheat.Value)}
              value={valueInput}
              onChange={e => setValueInput(e.target.value)}
              disabled={disabled || cheat.Enabled}
            />
          </div>
        )}
        {cheat.Input && isClamp && (
          <div className="cheat-input-row">
            <span className="threshold-label">{behavior === 'clamp_min' ? 'floor' : 'ceil'}</span>
            <input
              type="number"
              className="cheat-input"
              placeholder={String(cheat.Value)}
              value={valueInput}
              onChange={e => setValueInput(e.target.value)}
              disabled={disabled || cheat.Enabled}
            />
            {cheat.Enabled && (
              <button className="apply-btn" onClick={handleApply} disabled={disabled}>Apply</button>
            )}
          </div>
        )}
        {cheat.Input && !isThreshold && !isClamp && (
          <div className="cheat-input-row">
            <input
              type="number"
              className="cheat-input"
              placeholder="value"
              value={valueInput}
              onChange={e => setValueInput(e.target.value)}
              disabled={disabled}
            />
            {cheat.Enabled && (
              <button className="apply-btn" onClick={handleApply} disabled={disabled || !valueInput}>
                Apply
              </button>
            )}
          </div>
        )}
        <Toggle
          checked={cheat.Enabled}
          disabled={disabled || !canEnable}
          onChange={handleToggle}
        />
      </div>
    </div>
  )
}

// ── Game Tile ─────────────────────────────────────────────────────────────────

// tileColor derives a stable solid dark color from a game name, for the no-art fallback.
function tileColor(name) {
  let h = 0
  for (let i = 0; i < (name || '').length; i++) h = (h * 31 + name.charCodeAt(i)) % 360
  return `hsl(${h} 12% 16%)` // Solid matte carbon color with a very subtle brand tint
}

function GameTile({ trainer, onSelect }) {
  const [art, setArt] = useState(null) // null = loading, '' = no art, else data URI

  useEffect(() => {
    let alive = true
    TrainerImage(trainer.Filename)
      .then(d => { if (alive) setArt(d || '') })
      .catch(() => { if (alive) setArt('') })
    return () => { alive = false }
  }, [trainer.Filename])

  return (
    <button className="game-tile" onClick={onSelect}>
      <div className="game-tile-art">
        {art ? (
          <img src={art} alt={trainer.Game} />
        ) : (
          <div className="game-tile-fallback" style={{ background: tileColor(trainer.Game) }}>
            {(trainer.Game || '?').trim().charAt(0).toUpperCase()}
          </div>
        )}
      </div>
      <div className="game-tile-name">{trainer.Game}</div>
      <div className="game-tile-meta">{trainer.Exe} · v{trainer.Version}</div>
    </button>
  )
}

// ── TRAINER TAB ───────────────────────────────────────────────────────────────

function TrainerTab({ status, setStatus }) {
  const [list, setList] = useState([])
  const [loadingList, setLoadingList] = useState(false)
  const [listErr, setListErr] = useState(null)

  const [connecting, setConnecting] = useState(false)
  const [toggling, setToggling] = useState(-1) // index of cheat being toggled
  const [actionErr, setActionErr] = useState(null)

  const [posterArt, setPosterArt] = useState(null)

  // Load trainer list on mount
  useEffect(() => {
    setLoadingList(true)
    ListTrainers()
      .then(r => setList(r || []))
      .catch(e => setListErr(String(e)))
      .finally(() => setLoadingList(false))
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

  // Load cover art when selected trainer changes
  useEffect(() => {
    if (status) {
      TrainerImage(status.Filename)
        .then(d => setPosterArt(d || ''))
        .catch(() => setPosterArt(''))
    } else {
      setPosterArt(null)
    }
  }, [status?.Filename])

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

  const disconnect = useCallback(async () => {
    try {
      const s = await DisconnectTrainer()
      setStatus(s)
    } catch (e) {
      setActionErr(String(e))
    }
  }, [])

  const toggleCheat = useCallback(async (idx, enable, userValue = 0, userTrigger = 0) => {
    setToggling(idx)
    setActionErr(null)
    try {
      const s = await ToggleCheat(idx, enable, userValue, userTrigger)
      setStatus(s)
    } catch (e) {
      setActionErr(String(e))
    } finally {
      setToggling(-1)
    }
  }, [])

  return (
    <div className="trainer-tab">
      {!status ? (
        /* Gallery — cover-art grid of trainers */
        <div className="gallery">
          <div className="gallery-header">
            <span className="gallery-title">Installed Trainers</span>
            <button
              className="folder-btn"
              title="Open trainer folder"
              onClick={async () => {
                const dir = await UserTrainerDir()
                OpenFolder(dir)
              }}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
              </svg>
              Trainer Directory
            </button>
          </div>
          {loadingList && <div className="empty"><Spinner /></div>}
          {listErr && <div className="panel-error">{listErr}</div>}
          {!loadingList && list.length === 0 && !listErr && (
            <div className="empty">No trainers found</div>
          )}
          <div className="gallery-grid">
            {list.map(t => (
              <GameTile key={t.Filename} trainer={t} onSelect={() => selectTrainer(t.Filename)} />
            ))}
          </div>
        </div>
      ) : (
        /* Detail — WeMod Style Dashboard Layout */
        <main className="trainer-detail">
          <button className="back-btn" onClick={() => setStatus(null)}>
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="19" y1="12" x2="5" y2="12"></line>
              <polyline points="12 19 5 12 12 5"></polyline>
            </svg>
            Back to Trainers
          </button>

          <div className="trainer-dashboard-layout">
            {/* Left Cover Art Column */}
            <section className="trainer-left-col">
              <div className="trainer-poster-container">
                {posterArt ? (
                  <img src={posterArt} alt={status.Game} />
                ) : (
                  <div className="game-tile-fallback" style={{ background: tileColor(status.Game) }}>
                    {(status.Game || '?').trim().charAt(0).toUpperCase()}
                  </div>
                )}
              </div>
            </section>

            {/* Right Cheats Dashboard Column */}
            <section className="trainer-right-col">
              <div className="trainer-header">
                <div className="trainer-title">
                  <h1>{status.Game}</h1>
                  <span className="trainer-meta">{status.Exe} · Version {status.Version}</span>
                </div>
                <div className="trainer-connect">
                  {status.Connected ? (
                    <span className="badge running">Connected (PID {status.PID})</span>
                  ) : (
                    <span className="badge stopped">Disconnected</span>
                  )}
                  <button className="connect-btn" onClick={connect} disabled={connecting}>
                    {connecting ? <Spinner /> : status.Connected ? 'Reconnect' : 'Connect Game'}
                  </button>
                  {status.Connected && (
                    <button className="disconnect-btn" onClick={disconnect}>
                      Disconnect
                    </button>
                  )}
                </div>
              </div>

              {actionErr && <div className="action-error">{actionErr}</div>}

              <div className="cheat-list">
                {(status.Cheats || []).map((cheat, idx) => (
                  <CheatCard
                    key={idx}
                    cheat={cheat}
                    disabled={!status.Connected || toggling === idx}
                    onToggle={(on, userValue, userTrigger) => toggleCheat(idx, on, userValue, userTrigger)}
                  />
                ))}
              </div>
            </section>
          </div>
        </main>
      )}
    </div>
  )
}

// ── DEV MODE TAB ──────────────────────────────────────────────────────────────

// ── Pointer Chain Builder ─────────────────────────────────────────────────────

function PointerChainBuilder({ attachedPID }) {
  const [baseOffset, setBaseOffset] = useState('')
  const [offsets, setOffsets] = useState([''])
  const [result, setResult] = useState(null)
  const [walking, setWalking] = useState(false)
  const [err, setErr] = useState(null)

  const updateOffset = (i, val) => {
    setOffsets(prev => { const next = [...prev]; next[i] = val; return next })
  }

  const addOffset = () => setOffsets(prev => [...prev, ''])
  const removeOffset = (i) => setOffsets(prev => prev.filter((_, j) => j !== i))

  const walk = async () => {
    if (!baseOffset) return
    setWalking(true)
    setErr(null)
    setResult(null)
    try {
      const r = await WalkPointerChain(baseOffset, offsets.filter(o => o.trim() !== ''))
      if (r.Err) {
        setErr(r.Err)
      } else {
        setResult(r)
      }
    } catch (e) {
      setErr(String(e))
    } finally {
      setWalking(false)
    }
  }

  const copyChain = () => {
    if (!result) return
    const snippet = JSON.stringify({
      base_offset: baseOffset,
      offsets: offsets.filter(o => o.trim() !== ''),
    }, null, 2)
    navigator.clipboard.writeText(snippet)
  }

  return (
    <section className="dev-card dev-card-ptr-chain">
      <div className="dev-card-header"><span>Pointer Chain Builder</span></div>
      <div className="ptr-chain-hint">
        Enter a base offset (from module base) and pointer offsets to walk a chain and resolve the final address.
      </div>

      <div className="ptr-chain-row">
        <span className="ptr-chain-label">Base offset</span>
        <input
          className="ptr-chain-input"
          placeholder="e.g. 0x15c344"
          value={baseOffset}
          onChange={e => setBaseOffset(e.target.value)}
          disabled={walking}
        />
      </div>

      <div className="ptr-chain-offsets">
        <div className="ptr-chain-offsets-hdr">
          <span className="ptr-chain-label">Pointer offsets</span>
          <button className="ptr-chain-add-btn" onClick={addOffset} disabled={walking}>+ Add</button>
        </div>
        {offsets.map((off, i) => (
          <div key={i} className="ptr-chain-offset-row">
            <span className="ptr-chain-step-num">[{i}]</span>
            <input
              className="ptr-chain-input"
              placeholder="e.g. 0x40"
              value={off}
              onChange={e => updateOffset(i, e.target.value)}
              disabled={walking}
            />
            {offsets.length > 1 && (
              <button className="ptr-chain-remove-btn" onClick={() => removeOffset(i)} disabled={walking}>✕</button>
            )}
          </div>
        ))}
      </div>

      <div className="ptr-chain-actions">
        <button
          className="ptr-chain-walk-btn"
          onClick={walk}
          disabled={!attachedPID || !baseOffset || walking}
        >
          {walking ? <Spinner /> : 'Walk Chain'}
        </button>
      </div>

      {err && <div className="ptr-chain-error">{err}</div>}

      {result && (
        <div className="ptr-chain-result">
          <div className="ptr-chain-result-hdr">
            <span>Result</span>
            <button className="copy-btn" onClick={copyChain}>Copy JSON</button>
          </div>
          <div className="ptr-chain-result-start">
            Start: <code>{result.StartAddr}</code>
          </div>
          {result.Steps.map((step, i) => (
            <div key={i} className="ptr-chain-step">
              <span className="ptr-chain-step-num">[{i}]</span>
              <span className="ptr-chain-step-off">+{step.Offset}</span>
              <span className="ptr-chain-step-arrow">→</span>
              <code className="ptr-chain-step-addr">{step.ReadAddr}</code>
              <span className="ptr-chain-step-arrow">→</span>
              <code className="ptr-chain-step-ptr">{step.PointerVal}</code>
            </div>
          ))}
          <div className="ptr-chain-final">
            <span>Final address:</span>
            <code>{result.FinalAddr}</code>
          </div>
          <div className="ptr-chain-values">
            <span className="type-tag">int32</span>
            <code>{result.FinalInt32}</code>
            <span className="type-tag" style={{marginLeft:8}}>float32</span>
            <code>{result.FinalFloat32?.toFixed(4)}</code>
          </div>
        </div>
      )}
    </section>
  )
}

function getScanHint(history, count, attached, scanning, scanType) {
  if (!attached) return 'Select a process from the left to begin.'
  if (scanning) return scanType === 'all'
    ? 'Scanning all types in parallel — large processes may take a moment...'
    : 'Scan in progress — depending on the game this can take a moment...'
  if (history.length === 0) return 'Enter the current in-game value and click Scan.'
  if (count > 10000) return `${count.toLocaleString()} candidates — way too many. Change the value in-game, then use Narrow or a mode button.`
  if (count > 500) return `${count.toLocaleString()} candidates. Change the value in-game, then narrow it down.`
  if (count > 50) return `Getting closer — ${count} candidates. Keep changing the value in-game and narrowing.`
  if (count > 1) return `Almost there — ${count} candidates left. One more change should isolate it.`
  if (count === 1) return '✓ Found it! Verify with Test Freeze, then copy the trainer entry.'
  return 'No matches. The value type might be wrong (try float32 in the editor), or start a fresh scan.'
}

function DevModeTab() {
  const [showWarning, setShowWarning] = useState(() => localStorage.getItem(WARN_DISMISSED_KEY) !== 'true')
  const [neverShow, setNeverShow] = useState(false)

  const dismissWarning = () => {
    if (neverShow) localStorage.setItem(WARN_DISMISSED_KEY, 'true')
    setShowWarning(false)
  }

  // Process state
  const [attachedPID, setAttachedPID] = useState(0)
  const [attachedName, setAttachedName] = useState('')
  const [moduleBase, setModuleBase] = useState('')
  const [procs, setProcs] = useState([])
  const [filter, setFilter] = useState('')
  const [sortBy, setSortBy] = useState('pid-asc')
  const [loadingProcs, setLoadingProcs] = useState(false)
  const [procErr, setProcErr] = useState(null)

  // Scanner state
  const [addresses, setAddresses] = useState([])
  const [scanVal, setScanVal] = useState('')
  const [scanType, setScanType] = useState('int32') // 'int32', 'float32', or 'all'
  const [allTypesResults, setAllTypesResults] = useState(null) // only set in 'all' mode
  const [scanning, setScanning] = useState(false)
  const [scanHistory, setScanHistory] = useState([]) // timeline entries
  const [selectedAddr, setSelectedAddr] = useState(null)

  // Address inspector state (multi-type + test freeze)
  const [allTypes, setAllTypes] = useState(null)
  const [freezeVal, setFreezeVal] = useState('')
  const [freezeActive, setFreezeActive] = useState(false)
  const [freezeMsg, setFreezeMsg] = useState(null)

  // Add-to-trainer panel (shown when 1 result)
  const [trainerName, setTrainerName] = useState('')
  const [trainerType, setTrainerType] = useState('int32')
  const [trainerValue, setTrainerValue] = useState('9999')
  const [copiedJson, setCopiedJson] = useState(false)

  // Manual editor state
  const [writeAddr, setWriteAddr] = useState('')
  const [writeVal, setWriteVal] = useState('')
  const [readResult, setReadResult] = useState(null)
  const [writeMsg, setWriteMsg] = useState(null)
  const [clampWarning, setClampWarning] = useState(null) // { field, msg }

  const [statusMsg, setStatusMsg] = useState(null)
  const showStatus = (text, isError = false) => setStatusMsg({ text, isError })

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

  const detach = useCallback(async () => {
    await StopTestFreeze().catch(() => {})
    setFreezeActive(false)
    setAttachedPID(0)
    setAttachedName('')
    setModuleBase('')
    setAddresses([])
    setScanHistory([])
    setSelectedAddr(null)
    setAllTypes(null)
    showStatus('Detached.')
  }, [])

  const attach = useCallback(async (pid, name) => {
    // Stop any running test freeze before switching processes
    await StopTestFreeze().catch(() => {})
    setFreezeActive(false)
    try {
      await AttachProcess(pid)
      setAttachedPID(pid)
      setAttachedName(name)
      setAddresses([])
      setScanHistory([])
      setSelectedAddr(null)
      setAllTypes(null)
      setModuleBase('')
      showStatus(`Attached to ${name} (PID ${pid})`)
      const base = await GetModuleBase(pid)
      setModuleBase(base || '')
    } catch (e) {
      showStatus(String(e), true)
    }
  }, [])

  const addHistory = (type, query, mode, count) => {
    setScanHistory(prev => [...prev, { id: prev.length + 1, type, query, mode, count }])
  }

  const doScan = useCallback(async () => {
    const num = parseFloat(scanVal)
    if (isNaN(num)) return
    setScanning(true)
    try {
      if (scanType === 'all') {
        const r = await ScanAllTypes(num)
        setAllTypesResults(r)
        const all = [...(r.Int32?.Addresses||[]), ...(r.Float32?.Addresses||[]), ...(r.Int64?.Addresses||[]), ...(r.Float64?.Addresses||[])]
        setAddresses(all)
        setSelectedAddr(null)
        setScanHistory(h => [...h, { type: 'scan-all', val: num, count: all.length, i32: r.Int32?.Count||0, f32: r.Float32?.Count||0, i64: r.Int64?.Count||0, f64: r.Float64?.Count||0 }])
        return
      }
      const r = scanType === 'float32' ? await ScanValueFloat(num) : await ScanValue(Math.trunc(num))
      setAllTypesResults(null)
      setAddresses(r.Addresses || [])
      setSelectedAddr(null)
      setAllTypes(null)
      setScanHistory([])
      addHistory('initial', num, null, r.Count)
      showStatus(`Scanned — ${r.Count} match${r.Count === 1 ? '' : 'es'}`)
    } catch (e) {
      showStatus(String(e), true)
    } finally {
      setScanning(false)
    }
  }, [scanVal, scanType])

  const doNarrow = useCallback(async () => {
    const num = parseFloat(scanVal)
    if (isNaN(num)) return
    setScanning(true)
    try {
      if (scanType === 'all') {
        const r = await NarrowAllTypes(num)
        setAllTypesResults(r)
        const all = [...(r.Int32?.Addresses||[]), ...(r.Float32?.Addresses||[]), ...(r.Int64?.Addresses||[]), ...(r.Float64?.Addresses||[])]
        setAddresses(all)
        setSelectedAddr(null)
        setScanHistory(h => [...h, { type: 'narrow-all', val: num, count: all.length, i32: r.Int32?.Count||0, f32: r.Float32?.Count||0, i64: r.Int64?.Count||0, f64: r.Float64?.Count||0 }])
        showStatus(`Narrowed — ${all.length} total`)
        return
      }
      const r = scanType === 'float32' ? await NarrowValueFloat(num) : await NarrowValue(Math.trunc(num))
      setAddresses(r.Addresses || [])
      setSelectedAddr(null)
      setAllTypes(null)
      addHistory('exact', num, null, r.Count)
      showStatus(`Narrowed — ${r.Count} match${r.Count === 1 ? '' : 'es'}`)
    } catch (e) {
      showStatus(String(e), true)
    } finally {
      setScanning(false)
    }
  }, [scanVal, scanType])

  const doModeScan = useCallback(async (mode) => {
    setScanning(true)
    try {
      if (scanType === 'all') {
        const r = await ScanByModeAll(mode)
        setAllTypesResults(r)
        const all = [...(r.Int32?.Addresses||[]), ...(r.Float32?.Addresses||[]), ...(r.Int64?.Addresses||[]), ...(r.Float64?.Addresses||[])]
        setAddresses(all)
        setSelectedAddr(null)
        setScanHistory(h => [...h, { type: 'mode-all', val: null, mode, count: all.length, i32: r.Int32?.Count||0, f32: r.Float32?.Count||0, i64: r.Int64?.Count||0, f64: r.Float64?.Count||0 }])
        showStatus(`${mode} — ${all.length} total`)
        return
      }
      const r = await ScanByMode(mode)
      setAddresses(r.Addresses || [])
      setSelectedAddr(null)
      setAllTypes(null)
      addHistory('mode', null, mode, r.Count)
      showStatus(`${mode.charAt(0).toUpperCase() + mode.slice(1)} — ${r.Count} match${r.Count === 1 ? '' : 'es'}`)
    } catch (e) {
      showStatus(String(e), true)
    } finally {
      setScanning(false)
    }
  }, [scanType])

  const selectAddr = useCallback(async (addr) => {
    setSelectedAddr(addr)
    setWriteAddr(addr)
    setAllTypes(null)
    setFreezeMsg(null)
    try {
      const t = await ReadAllTypes(addr)
      setAllTypes(t)
      setFreezeVal(String(t.Int32))
    } catch (_) {}
  }, [])

  const TYPE_BOUNDS = {
    'int32':   { min: -2147483648,          max: 2147483647,          int: true  },
    'int64':   { min: -9223372036854775808,  max: 9223372036854775807, int: true  },
    'float32': { min: -3.4028235e+38,        max: 3.4028235e+38,       int: false },
    'float64': { min: -Number.MAX_VALUE,     max: Number.MAX_VALUE,    int: false },
    'all':     { min: -2147483648,           max: 2147483647,          int: true  },
  }

  const clampToType = (raw, field) => {
    const bounds = TYPE_BOUNDS[scanType] || TYPE_BOUNDS['int32']
    const num = bounds.int ? parseInt(raw, 10) : parseFloat(raw)
    if (isNaN(num)) return raw
    if (num > bounds.max) {
      setClampWarning({ field, msg: `Clamped to ${scanType} max (${bounds.max})` })
      setTimeout(() => setClampWarning(null), 3000)
      return String(bounds.max)
    }
    if (num < bounds.min) {
      setClampWarning({ field, msg: `Clamped to ${scanType} min (${bounds.min})` })
      setTimeout(() => setClampWarning(null), 3000)
      return String(bounds.min)
    }
    setClampWarning(null)
    return raw
  }

  const startFreeze = useCallback(async () => {
    const num = parseInt(freezeVal, 10)
    if (isNaN(num) || !selectedAddr) return
    try {
      await TestFreeze(selectedAddr, num)
      setFreezeActive(true)
      setFreezeMsg({ text: `Freezing ${selectedAddr} at ${num}`, err: false })
    } catch (e) {
      setFreezeMsg({ text: String(e), err: true })
    }
  }, [selectedAddr, freezeVal])

  const stopFreeze = useCallback(async () => {
    await StopTestFreeze().catch(() => {})
    setFreezeActive(false)
    setFreezeMsg({ text: 'Freeze stopped — value restored to game control.', err: false })
  }, [])

  const copyTrainerJson = useCallback(() => {
    if (!addresses[0] || !moduleBase) return
    const addr = BigInt(addresses[0])
    const base = BigInt(moduleBase)
    const offset = '0x' + (addr - base).toString(16)
    const snippet = JSON.stringify({
      name: trainerName || 'My Cheat',
      description: '',
      type: trainerType,
      base_offset: offset,
      value: parseFloat(trainerValue) || 0,
    }, null, 2)
    navigator.clipboard.writeText(snippet)
    setCopiedJson(true)
    setTimeout(() => setCopiedJson(false), 2000)
  }, [addresses, moduleBase, trainerName, trainerType, trainerValue])

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

  const SORT_CYCLE = ['pid-asc', 'pid-desc', 'name-asc', 'name-desc']
  const SORT_LABEL = { 'pid-asc': 'PID ↑', 'pid-desc': 'PID ↓', 'name-asc': 'A→Z', 'name-desc': 'Z→A' }
  const cycleSort = () => setSortBy(s => SORT_CYCLE[(SORT_CYCLE.indexOf(s) + 1) % SORT_CYCLE.length])

  const filtered = procs
    .filter(p =>
      (p.Name || '').toLowerCase().includes(filter.toLowerCase()) ||
      String(p.PID).includes(filter)
    )
    .sort((a, b) => {
      if (sortBy === 'pid-asc')  return a.PID - b.PID
      if (sortBy === 'pid-desc') return b.PID - a.PID
      if (sortBy === 'name-asc') return (a.Name || '').localeCompare(b.Name || '')
      return (b.Name || '').localeCompare(a.Name || '')
    })

  // Compute offset when exactly 1 address found
  const singleOffset = addresses.length === 1 && moduleBase ? (() => {
    try {
      return '0x' + (BigInt(addresses[0]) - BigInt(moduleBase)).toString(16)
    } catch (_) { return null }
  })() : null

  useEffect(() => { refreshProcs() }, [])

  // Clean up test freeze on unmount
  useEffect(() => () => { StopTestFreeze().catch(() => {}) }, [])

  const hint = getScanHint(scanHistory, addresses.length, attachedPID > 0, scanning, scanType)
  const hasHistory = scanHistory.length > 0

  return (
    <div className="dev-tab">
      {/* Safety Warning Banner */}
      {showWarning && (
        <div className="dev-warning-banner">
          <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" style={{flexShrink:0,marginTop:2}}>
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          <div className="dev-warning-body">
            <span><strong>Dev Mode is for single-player game memory research only.</strong> Writing to the wrong process can crash it. Only attach to games you own.</span>
            <label className="dev-warning-checkbox">
              <input type="checkbox" checked={neverShow} onChange={e => setNeverShow(e.target.checked)} />
              Never show this again
            </label>
          </div>
          <button className="dev-warning-close" onClick={dismissWarning} title="Dismiss">
            <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      )}

      {/* Status bar */}
      {statusMsg && (
        <div className={`dev-status ${statusMsg.isError ? 'error' : 'ok'}`}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            {statusMsg.isError
              ? <><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></>
              : <polyline points="20 6 9 17 4 12"/>}
          </svg>
          {statusMsg.text}
        </div>
      )}

      <div className="dev-layout">
        {/* ── Left: Process Panel ── */}
        <section className="dev-card dev-card-process">
          <div className="dev-card-header">
            <span>Process</span>
            <button onClick={refreshProcs} disabled={loadingProcs}>
              {loadingProcs ? <Spinner /> : 'Refresh'}
            </button>
          </div>
          <div className="dev-filter-row">
            <input
              className="dev-filter"
              placeholder="Search..."
              value={filter}
              onChange={e => setFilter(e.target.value)}
            />
            <button className="dev-sort-btn" onClick={cycleSort} title="Cycle sort order">
              {SORT_LABEL[sortBy]}
            </button>
          </div>
          {attachedPID > 0 && (
            <div className="dev-attached-badge">
              <span className="pulse-dot" />
              <span className="dev-attached-label">{attachedName} · PID {attachedPID}</span>
              {moduleBase && <span className="dev-base-tag">{moduleBase}</span>}
              <button className="dev-detach-btn" onClick={detach} title="Detach process">✕</button>
            </div>
          )}
          {procErr && <div className="panel-error">{procErr}</div>}
          <div className="dev-proc-list">
            {filtered.length === 0 && !procErr && (
              <div className="empty">Click Refresh to load</div>
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

        {/* ── Center: Scanner Panel ── */}
        <section className="dev-card dev-card-scanner">
          <div className="dev-card-header"><span>Memory Scanner</span></div>

          {/* Contextual hint */}
          <div className={`scan-hint ${addresses.length === 1 ? 'success' : ''} ${scanning ? 'scanning' : ''}`}>
            {scanning && <Spinner />}
            {hint}
          </div>

          {/* Scan type toggle */}
          <div className="scan-type-row">
            <span className="scan-mode-label">Value type:</span>
            <div className="scan-type-toggle">
              <button
                className={`scan-type-btn ${scanType === 'int32' ? 'active' : ''}`}
                onClick={() => { setScanType('int32'); setScanHistory([]); setAddresses([]); setAllTypesResults(null) }}
                disabled={scanning}
              >int32</button>
              <button
                className={`scan-type-btn ${scanType === 'float32' ? 'active' : ''}`}
                onClick={() => { setScanType('float32'); setScanHistory([]); setAddresses([]); setAllTypesResults(null) }}
                disabled={scanning}
              >float32</button>
              <button
                className={`scan-type-btn ${scanType === 'all' ? 'active' : ''}`}
                onClick={() => { setScanType('all'); setScanHistory([]); setAddresses([]); setAllTypesResults(null) }}
                disabled={scanning}
              >all types</button>
            </div>
          </div>

          {/* Scan controls */}
          <div className="dev-row">
            <input
              type="number"
              placeholder={`Known value (${scanType})...`}
              value={scanVal}
              onChange={e => setScanVal(e.target.value)}
              disabled={!attachedPID || scanning}
              onKeyDown={e => e.key === 'Enter' && (hasHistory ? doNarrow() : doScan())}
            />
            <button
              onClick={doScan}
              disabled={!attachedPID || scanning || !scanVal}
              title="Start a fresh scan for this value"
            >
              {scanning ? <Spinner /> : 'Scan'}
            </button>
            <button
              onClick={doNarrow}
              disabled={!attachedPID || scanning || !scanVal || !hasHistory}
              className="secondary"
              title="Narrow to addresses that now equal this value"
            >
              Narrow
            </button>
          </div>

          {/* Mode scan buttons */}
          <div className="scan-mode-row">
            <span className="scan-mode-label">Mode scan:</span>
            {['increased', 'decreased', 'changed', 'unchanged'].map(mode => (
              <button
                key={mode}
                className={`scan-mode-btn scan-mode-${mode}`}
                onClick={() => doModeScan(mode)}
                disabled={!attachedPID || scanning || !hasHistory}
                title={`Keep addresses whose value has ${mode} since the last scan`}
              >
                {mode.charAt(0).toUpperCase() + mode.slice(1)}
              </button>
            ))}
          </div>

          {/* Scan history timeline */}
          {hasHistory && (
            <div className="scan-history">
              {scanHistory.map((entry, i) => (
                <div key={entry.id} className={`scan-history-entry ${i === scanHistory.length - 1 ? 'latest' : ''}`}>
                  <span className="scan-history-step">{entry.id}</span>
                  <span className="scan-history-desc">
                    {entry.type === 'initial' && `Scan for ${entry.query}`}
                    {entry.type === 'exact' && `Narrow → ${entry.query}`}
                    {entry.type === 'mode' && entry.mode.charAt(0).toUpperCase() + entry.mode.slice(1)}
                  </span>
                  <span className={`scan-history-count ${entry.count <= 1 ? 'done' : ''}`}>
                    {entry.count === 1 ? '✓ 1' : entry.count.toLocaleString()}
                  </span>
                </div>
              ))}
            </div>
          )}

          {/* Results */}
          <div className="dev-results-hdr">
            <span>
              {hasHistory
                ? `${addresses.length} address${addresses.length === 1 ? '' : 'es'}${addresses.length > 200 ? ' (first 200 shown)' : ''}`
                : 'No scan yet'}
            </span>
            {addresses.length === 1 && <span className="found-badge">Found!</span>}
          </div>

          <div className="dev-addr-list">
            {scanType === 'all' && allTypesResults ? (
              [['int32', allTypesResults.Int32], ['float32', allTypesResults.Float32], ['int64', allTypesResults.Int64], ['float64', allTypesResults.Float64]].map(([label, res]) =>
                res?.Count > 0 && (
                  <div key={label} className="all-types-group">
                    <div className="all-types-group-header">
                      <span className="all-types-label">{label}</span>
                      <span className="all-types-count">{res.Count}</span>
                    </div>
                    {(res.Addresses.length > 50 ? res.Addresses.slice(0, 50) : res.Addresses).map(addr => (
                      <div
                        key={`${label}-${addr}`}
                        className={`dev-addr-row ${addr === selectedAddr ? 'selected' : ''}`}
                        onClick={() => selectAddr(addr)}
                      >
                        {addr}
                        <span className="addr-type-tag">{label}</span>
                        {addr === selectedAddr && <span className="addr-selected-tag">selected</span>}
                      </div>
                    ))}
                    {res.Addresses.length > 50 && <div className="addr-overflow">+{res.Addresses.length - 50} more</div>}
                  </div>
                )
              )
            ) : (
              (addresses.length > 200 ? addresses.slice(0, 200) : addresses).map(addr => (
                <div
                  key={addr}
                  className={`dev-addr-row ${addr === selectedAddr ? 'selected' : ''}`}
                  onClick={() => selectAddr(addr)}
                >
                  {addr}
                  {addr === selectedAddr && <span className="addr-selected-tag">selected</span>}
                </div>
              ))
            )}
          </div>
        </section>

        {/* ── Right: Inspector + Editor ── */}
        <div className="dev-right">
          {/* Address Inspector */}
          {selectedAddr && (
            <section className="dev-card dev-card-inspector">
              <div className="dev-card-header"><span>Address Inspector</span></div>

              <div className="inspector-addr"><code>{selectedAddr}</code></div>

              {/* Multi-type value display */}
              {allTypes && allTypes.Valid && (
                <div className="inspector-types">
                  <div className="inspector-type-row">
                    <span className="type-tag">int32</span>
                    <code>{allTypes.Int32}</code>
                  </div>
                  <div className="inspector-type-row">
                    <span className="type-tag">float32</span>
                    <code>{allTypes.Float32.toFixed(4)}</code>
                  </div>
                  <div className="inspector-type-row">
                    <span className="type-tag">int64</span>
                    <code>{allTypes.Int64}</code>
                  </div>
                </div>
              )}

              {/* Test freeze */}
              <div className="inspector-freeze">
                <span className="inspector-section-label">Test Freeze</span>
                <div className="dev-row">
                  <input
                    type="number"
                    placeholder="Freeze at value..."
                    value={freezeVal}
                    onChange={e => setFreezeVal(clampToType(e.target.value, 'freeze'))}
                    disabled={freezeActive}
                  />
                  {!freezeActive ? (
                    <button
                      className="freeze-btn"
                      onClick={startFreeze}
                      disabled={!freezeVal}
                    >
                      Freeze
                    </button>
                  ) : (
                    <button className="freeze-stop-btn" onClick={stopFreeze}>
                      Stop
                    </button>
                  )}
                </div>
                {freezeActive && (
                  <div className="freeze-active-indicator">
                    <span className="pulse-dot green" /> Actively freezing
                  </div>
                )}
                {clampWarning?.field === 'freeze' && (
                  <div className="clamp-warning">{clampWarning.msg}</div>
                )}
                {freezeMsg && (
                  <div className={`dev-write-msg ${freezeMsg.err ? 'error' : 'ok'}`}>
                    {freezeMsg.text}
                  </div>
                )}
              </div>
            </section>
          )}

          {/* Add to Trainer panel — only when exactly 1 result */}
          {addresses.length === 1 && singleOffset && (
            <section className="dev-card dev-card-trainer-export">
              <div className="dev-card-header"><span>Add to Trainer</span></div>
              <div className="trainer-export-offset">
                Offset: <code>{singleOffset}</code>
                <button className="copy-btn" onClick={() => navigator.clipboard.writeText(singleOffset)}>Copy</button>
              </div>
              <div className="dev-row">
                <input
                  placeholder="Cheat name..."
                  value={trainerName}
                  onChange={e => setTrainerName(e.target.value)}
                />
                <select
                  className="trainer-type-select"
                  value={trainerType}
                  onChange={e => setTrainerType(e.target.value)}
                >
                  <option value="int32">int32</option>
                  <option value="float32">float32</option>
                  <option value="int64">int64</option>
                </select>
              </div>
              <div className="dev-row">
                <input
                  type="number"
                  placeholder="Cheat value..."
                  value={trainerValue}
                  onChange={e => setTrainerValue(e.target.value)}
                />
                <button className={`copy-json-btn ${copiedJson ? 'copied' : ''}`} onClick={copyTrainerJson}>
                  {copiedJson ? '✓ Copied!' : 'Copy JSON'}
                </button>
              </div>
            </section>
          )}

          {/* Pointer Chain Builder */}
          <PointerChainBuilder attachedPID={attachedPID} />

          {/* Memory Editor */}
          <section className="dev-card dev-card-writer">
            <div className="dev-card-header"><span>Memory Editor</span></div>
            <div className="dev-row">
              <input
                placeholder="Address (0x...)"
                value={writeAddr}
                onChange={e => { setWriteAddr(e.target.value); setReadResult(null) }}
                disabled={!attachedPID}
              />
              <button onClick={doRead} disabled={!attachedPID || !writeAddr} className="secondary">
                Read
              </button>
            </div>
            {readResult !== null && (
              <div className="dev-current-val">Current Value: <strong>{readResult}</strong></div>
            )}
            <div className="dev-row">
              <input
                type="number"
                placeholder="New value..."
                value={writeVal}
                onChange={e => setWriteVal(clampToType(e.target.value, 'write'))}
                disabled={!attachedPID}
              />
              <button onClick={doWrite} disabled={!attachedPID || !writeAddr || !writeVal} className="danger">
                Write
              </button>
            </div>
            {clampWarning?.field === 'write' && (
              <div className="clamp-warning">{clampWarning.msg}</div>
            )}
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

// ── Shared constants ──────────────────────────────────────────────────────────

const WARN_DISMISSED_KEY = 'devmode_warning_dismissed'

// ── Scale Slider ──────────────────────────────────────────────────────────────

const SCALE_TICKS = [
  { value: 0.75, label: '75%' },
  { value: 1.00, label: '100%' },
  { value: 1.25, label: '125%' },
  { value: 1.50, label: '150%' },
]
const SCALE_MIN = 0.75
const SCALE_MAX = 1.50
const SCALE_STEP = 0.05

function ScaleSlider({ value, onChange }) {
  const pct = ((value - SCALE_MIN) / (SCALE_MAX - SCALE_MIN)) * 100

  return (
    <div className="scale-slider-wrap">
      {/* Track + thumb */}
      <div className="scale-slider-track-row">
        <div className="scale-slider-track">
          <div className="scale-slider-fill" style={{ width: `${pct}%` }} />
          <input
            type="range"
            className="scale-slider-input"
            min={SCALE_MIN}
            max={SCALE_MAX}
            step={SCALE_STEP}
            value={value}
            onChange={e => onChange(parseFloat(e.target.value))}
          />
        </div>
      </div>

      {/* Tick marks + labels */}
      <div className="scale-slider-ticks">
        {SCALE_TICKS.map(tick => {
          const pos = ((tick.value - SCALE_MIN) / (SCALE_MAX - SCALE_MIN)) * 100
          const active = Math.abs(value - tick.value) < 0.001
          return (
            <div
              key={tick.value}
              className={`scale-tick ${active ? 'active' : ''}`}
              style={{ left: `${pos}%` }}
              onClick={() => onChange(tick.value)}
              title={tick.label}
            >
              <div className="scale-tick-mark" />
              <span className="scale-tick-label">{tick.label}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ── SETTINGS TAB ──────────────────────────────────────────────────────────────

function SettingsTab({ uiScale, onScaleChange }) {
  const [settings, setSettings] = useState(null)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveMsg, setSaveMsg] = useState(null)
  const [warningsEnabled, setWarningsEnabled] = useState(
    localStorage.getItem(WARN_DISMISSED_KEY) !== 'true'
  )

  useEffect(() => {
    GetSettings().then(s => setSettings(s)).catch(() => {})
  }, [])

  const update = (key, val) => {
    setSettings(prev => ({ ...prev, [key]: val }))
    setDirty(true)
    setSaveMsg(null)
  }

  const save = async () => {
    setSaving(true)
    setSaveMsg(null)
    try {
      await SaveSettings(settings)
      setDirty(false)
      setSaveMsg({ text: 'Settings saved.', err: false })
    } catch (e) {
      setSaveMsg({ text: String(e), err: true })
    } finally {
      setSaving(false)
    }
  }

  const pickDir = async () => {
    const dir = await PickTrainerDir()
    if (dir) update('TrainerDir', dir)
  }

  const clearDir = () => update('TrainerDir', '')

  const toggleWarnings = () => {
    if (warningsEnabled) {
      localStorage.setItem(WARN_DISMISSED_KEY, 'true')
      setWarningsEnabled(false)
    } else {
      localStorage.removeItem(WARN_DISMISSED_KEY)
      setWarningsEnabled(true)
    }
  }

  if (!settings) return <div className="settings-loading"><Spinner /></div>

  return (
    <div className="settings-tab">
      <div className="settings-header">
        <h1>Settings</h1>
        <p>Configure FreeMod behavior. Changes are saved per-machine.</p>
      </div>

      {/* ── Interface ── */}
      <section className="settings-section">
        <div className="settings-section-title">Interface</div>
        <div className="settings-row settings-row-col">
          <div className="settings-row-info">
            <span className="settings-row-label">
              UI Scale
              <code className="settings-row-value">{Math.round(uiScale * 100)}%</code>
            </span>
            <span className="settings-row-desc">
              Resize all UI elements. Takes effect immediately — no restart needed.
            </span>
          </div>
          <ScaleSlider value={uiScale} onChange={onScaleChange} />
        </div>

        <div className="settings-row">
          <div className="settings-row-info">
            <span className="settings-row-label">Launch fullscreen</span>
            <span className="settings-row-desc">
              Open FreeMod in fullscreen mode on every launch. Toggle applies immediately.
            </span>
          </div>
          <button
            className={`settings-toggle ${settings.LaunchFullscreen ? 'on' : 'off'}`}
            onClick={() => {
              const next = !settings.LaunchFullscreen
              update('LaunchFullscreen', next)
              SetFullscreen(next)
            }}
          >
            <span className="settings-toggle-knob" />
          </button>
        </div>
      </section>

      {/* ── General ── */}
      <section className="settings-section">
        <div className="settings-section-title">General</div>

        <div className="settings-row">
          <div className="settings-row-info">
            <span className="settings-row-label">Auto-connect</span>
            <span className="settings-row-desc">
              Automatically connect when a loaded game's process starts. Disable for manual control.
            </span>
          </div>
          <button
            className={`settings-toggle ${settings.AutoConnect ? 'on' : 'off'}`}
            onClick={() => update('AutoConnect', !settings.AutoConnect)}
          >
            <span className="settings-toggle-knob" />
          </button>
        </div>
      </section>

      {/* ── Performance ── */}
      <section className="settings-section">
        <div className="settings-section-title">Performance</div>

        <div className="settings-row settings-row-col">
          <div className="settings-row-info">
            <span className="settings-row-label">
              Freeze interval
              <code className="settings-row-value">{settings.FreezeIntervalMs} ms</code>
            </span>
            <span className="settings-row-desc">
              How often frozen cheats re-write their value. Lower = more responsive, higher = less CPU. Range: 50–1000 ms.
            </span>
          </div>
          <input
            type="range"
            className="settings-slider"
            min={50}
            max={1000}
            step={25}
            value={settings.FreezeIntervalMs}
            onChange={e => update('FreezeIntervalMs', parseInt(e.target.value, 10))}
          />
          <div className="settings-slider-labels">
            <span>50 ms</span>
            <span>1000 ms</span>
          </div>
        </div>
      </section>

      {/* ── Directories ── */}
      <section className="settings-section">
        <div className="settings-section-title">Directories</div>

        <div className="settings-row settings-row-col">
          <div className="settings-row-info">
            <span className="settings-row-label">Custom trainer directory</span>
            <span className="settings-row-desc">
              Load user trainers from a custom folder instead of the default
              <code> ~/Library/Application Support/FreeMod/trainers/</code>.
              Leave blank to use the default.
            </span>
          </div>
          <div className="settings-dir-row">
            <input
              className="settings-dir-input"
              type="text"
              placeholder="Default (click Browse to change)"
              value={settings.TrainerDir}
              onChange={e => update('TrainerDir', e.target.value)}
            />
            <button className="settings-dir-btn" onClick={pickDir}>Browse</button>
            {settings.TrainerDir && (
              <button className="settings-dir-clear" onClick={clearDir} title="Reset to default">✕</button>
            )}
          </div>
        </div>
      </section>

      {/* ── Warnings ── */}
      <section className="settings-section">
        <div className="settings-section-title">Warnings</div>

        <div className="settings-row">
          <div className="settings-row-info">
            <span className="settings-row-label">Dev Mode safety warning</span>
            <span className="settings-row-desc">
              Show the "single-player only" banner at the top of the Memory Suite tab.
            </span>
          </div>
          <button
            className={`settings-toggle ${warningsEnabled ? 'on' : 'off'}`}
            onClick={toggleWarnings}
          >
            <span className="settings-toggle-knob" />
          </button>
        </div>
      </section>

      {/* ── Save ── */}
      <div className="settings-footer">
        {saveMsg && (
          <span className={`settings-save-msg ${saveMsg.err ? 'err' : 'ok'}`}>
            {saveMsg.text}
          </span>
        )}
        <button
          className="settings-save-btn"
          onClick={save}
          disabled={!dirty || saving}
        >
          {saving ? <Spinner /> : 'Save Changes'}
        </button>
      </div>
    </div>
  )
}

// ── Root App Layout ───────────────────────────────────────────────────────────

const UI_SCALE_KEY = 'ui_scale'
const THEME_KEY = 'theme'

export default function App() {
  const [tab, setTab] = useState('trainers')
  const [trainerStatus, setTrainerStatus] = useState(null)
  const [uiScale, setUiScale] = useState(() =>
    parseFloat(localStorage.getItem(UI_SCALE_KEY) || '1')
  )
  const [theme, setTheme] = useState(() =>
    localStorage.getItem(THEME_KEY) || 'dark'
  )

  const applyScale = (val) => {
    setUiScale(val)
    localStorage.setItem(UI_SCALE_KEY, val)
    document.documentElement.style.zoom = val
  }

  useEffect(() => {
    document.documentElement.style.zoom = uiScale
  }, [])

  const toggleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark'
    setTheme(next)
    localStorage.setItem(THEME_KEY, next)
  }

  // Track connected state globally to display a persistent notification widget in the sidebar
  const [globalGame, setGlobalGame] = useState(null)

  useEffect(() => {
    const id = setInterval(async () => {
      try {
        const s = await GetTrainerStatus()
        if (s && s.Game && s.Connected) {
          setGlobalGame({ name: s.Game, pid: s.PID, filename: s.Filename })
        } else {
          setGlobalGame(null)
        }
      } catch (_) {}
    }, 1000)
    return () => clearInterval(id)
  }, [])

  return (
    <ErrorBoundary>
      <div className="app" data-theme={theme}>
        {/* Premium WeMod style Sidebar */}
        <aside className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <svg className="logo-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
              </svg>
              <span>FreeMod</span>
            </div>
          </div>

          <nav className="sidebar-nav">
            <button
              className={`sidebar-btn ${tab === 'trainers' ? 'active' : ''}`}
              onClick={() => setTab('trainers')}
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"></rect>
                <line x1="7" y1="2" x2="7" y2="22"></line>
                <line x1="17" y1="2" x2="17" y2="22"></line>
                <line x1="2" y1="12" x2="22" y2="12"></line>
              </svg>
              Trainers Gallery
            </button>
            <button
              className={`sidebar-btn ${tab === 'dev' ? 'active' : ''}`}
              onClick={() => setTab('dev')}
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect>
                <rect x="9" y="9" width="6" height="6"></rect>
                <line x1="9" y1="1" x2="9" y2="4"></line>
                <line x1="15" y1="1" x2="15" y2="4"></line>
                <line x1="9" y1="20" x2="9" y2="23"></line>
                <line x1="15" y1="20" x2="15" y2="23"></line>
                <line x1="20" y1="9" x2="23" y2="9"></line>
                <line x1="20" y1="15" x2="23" y2="15"></line>
                <line x1="1" y1="9" x2="4" y2="9"></line>
                <line x1="1" y1="15" x2="4" y2="15"></line>
              </svg>
              Memory Suite
            </button>
            <button
              className={`sidebar-btn ${tab === 'settings' ? 'active' : ''}`}
              onClick={() => setTab('settings')}
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="3"></circle>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
              </svg>
              Settings
            </button>
          </nav>

          <div className="sidebar-footer">
            {/* Connection status notification in sidebar */}
            {globalGame && (
              <div className="status-widget">
                <span className="status-widget-title">Active Connection</span>
                <span className="status-widget-game">{globalGame.name}</span>
                <span className="status-widget-badge">Running (PID {globalGame.pid})</span>
              </div>
            )}
            
            <div className="theme-toggle-row">
              <button className={`theme-icon-btn sun ${theme === 'light' ? 'active' : ''}`} onClick={() => theme !== 'light' && toggleTheme()} title="Light mode">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
                </svg>
              </button>
              <button className={`theme-icon-btn moon ${theme === 'dark' ? 'active' : ''}`} onClick={() => theme !== 'dark' && toggleTheme()} title="Dark mode">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
                </svg>
              </button>
            </div>
            <button className="sidebar-kill-btn" onClick={() => KillApp()} title="Close application & deactivate all active cheats">
              Panic Close & Kill
            </button>
          </div>
        </aside>

        {/* Content viewport */}
        <div className="app-body">
          {tab === 'trainers' ? (
            <TrainerTab status={trainerStatus} setStatus={setTrainerStatus} />
          ) : tab === 'dev' ? (
            <DevModeTab />
          ) : (
            <SettingsTab uiScale={uiScale} onScaleChange={applyScale} />
          )}
        </div>
      </div>
    </ErrorBoundary>
  )
}
