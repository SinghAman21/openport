import type { Guide } from '@/lib/usecases'
import './usecase-visuals.css'

type Props = { guide: Guide; step: number; reducedMotion: boolean }
type VisualProps = { frame: Guide['frames'][number]; stage: number }

function Conflict({ frame, stage }: VisualProps) {
  return <div className="uv-conflict uv-scene" data-stage={stage}>
    <div className="uv-conflict-track"><div className="uv-conflict-app">APP<br /><strong>:3000</strong></div><span className="uv-conflict-barrier">{stage === 2 ? '✓' : '×'}</span><div className="uv-conflict-owner">{stage === 0 ? 'UNKNOWN OWNER' : stage === 1 ? 'node · PID 5678' : 'PORT AVAILABLE'}</div></div>
    <div className="uv-conflict-neighbors"><span>:5432 · {stage === 0 ? 'database' : 'protected'}</span><span>:6379 · redis · running</span></div>
    <p className="uv-scene-note">{frame.rows[0].detail}</p>
  </div>
}

function Stack({ frame, stage }: VisualProps) {
  return <div className="uv-stack uv-scene" data-stage={stage}>
    <div className="uv-stack-map">{frame.rows.map((row, i) => <div className="uv-stack-service" key={i}><span className="uv-stack-port">:{[3000, 5432, 6379][i]}</span><span className="uv-stack-connector" /><span className="uv-stack-alias">{stage === 0 ? ['?', '?', '?'][i] : ['frontend', 'database', 'cache'][i]}{stage === 2 && i === 1 ? ' · locked' : ''}</span></div>)}</div>
    <div className="uv-stack-file">{stage === 2 ? '.openport.yml  /  expected_ports + protected: [5432]' : stage === 1 ? 'Aliases view  /  names attached to listeners' : 'PORT MAP  /  unnamed'}</div>
  </div>
}

function FreePort({ frame, stage }: VisualProps) {
  return <div className="uv-free uv-scene" data-stage={stage}>
    <div className="uv-free-ruler">{frame.rows.map((row, i) => <div key={row.name} className={`uv-free-candidate uv-free-candidate-${i}`}><span className="uv-free-tick" /><strong>{row.name}</strong><small>{row.detail}</small></div>)}</div>
    <div className="uv-free-result">{stage === 0 ? 'NEXT ≠ FREE' : stage === 1 ? 'BIND TEST → PASS' : 'USE 4002  /  RESERVE IF NEEDED'}</div>
  </div>
}

function Blueprint({ frame, stage }: VisualProps) {
  return <div className="uv-blueprint uv-scene" data-stage={stage}>
    <div className="uv-blueprint-columns"><div><b>EXPECTED / web</b>{frame.rows.map((row, i) => <span key={i}>{row.name}<em>{[':3000', ':5432', ':9000'][i]}</em></span>)}</div><div><b>{stage === 0 ? 'ACTUAL / ?' : stage === 1 ? 'ACTUAL / scan' : 'REPORT / read only'}</b>{frame.rows.map((row, i) => <span key={i} className={i === 2 && stage > 0 ? 'uv-blueprint-missing' : ''}>{stage === 0 ? '—' : i === 2 ? stage === 1 ? 'not running' : 'start it yourself' : stage === 1 ? 'running' : 'leave running'}</span>)}</div></div>
    <div className="uv-blueprint-diff">{stage === 0 ? 'Define intent before comparing' : stage === 1 ? '− worker :9000 · missing' : 'No processes started or stopped'}</div>
  </div>
}

function Watch({ frame, stage }: VisualProps) {
  return <div className="uv-watch uv-scene" data-stage={stage}>
    <div className="uv-watch-line"><div className="uv-watch-node uv-watch-old">OLD PID</div><div className="uv-watch-node uv-watch-gap">RESTART</div><div className="uv-watch-node uv-watch-new">NEW PID 6721</div></div>
    <div className="uv-watch-playhead" /><div className="uv-watch-time"><span>LISTENING</span><span>REMOVED</span><span>ADDED</span></div>
    <div className="uv-watch-readout">:3000 / {frame.rows[0].detail}<br /><small>:5432 / {frame.rows[1].detail}</small></div>
  </div>
}

function Groups({ frame, stage }: VisualProps) {
  return <div className="uv-groups uv-scene" data-stage={stage}>
    <div className="uv-groups-cluster"><div className="uv-groups-branches" />{frame.rows.map((row, i) => <div className="uv-groups-member" key={i}><strong>{['frontend', 'api', 'worker'][i]}</strong><small>:{[3000, 8080, 9000][i]} {stage === 1 ? i === 2 ? '· review' : '· checked' : ''}</small></div>)}</div>
    <div className="uv-groups-base">{stage === 0 ? 'SEPARATE LISTENERS' : stage === 1 ? 'dev_stack  /  H · HEALTH CHECK' : 'dev_stack  /  x · REVIEW → CONFIRM'}</div>
    {stage === 2 && <div className="uv-groups-confirm">CONFIRMATION REQUIRED · PROTECTED PORTS STAY GUARDED</div>}
  </div>
}

function ProtectedPorts({ frame, stage }: VisualProps) {
  return <div className="uv-protected uv-scene" data-stage={stage}>
    <div className="uv-protected-gate"><div className="uv-protected-target">CLEANUP REQUEST <strong>→ :5432</strong></div><div className="uv-protected-shield">{stage === 0 ? '!' : '◆'}<small>{stage === 0 ? 'AT RISK' : 'GUARDED'}</small></div><div className="uv-protected-verdict">{stage === 0 ? 'NO GUARDRAIL' : stage === 1 ? 'PROTECT :22 + :5432' : 'KILL REFUSED'}</div></div>
    <p className="uv-protected-foot">{stage === 2 ? 'Override: UI only · two confirmations' : frame.rows.map((row) => `${row.name} ${row.detail}`).join('  /  ')}</p>
  </div>
}

const visuals: Record<string, (props: VisualProps) => React.JSX.Element> = {
  conflict: Conflict,
  stack: Stack,
  'free-port': FreePort,
  blueprint: Blueprint,
  'watch-mode': Watch,
  groups: Groups,
  'protected-ports': ProtectedPorts,
}

export default function UseCaseVisual({ guide, step, reducedMotion }: Props) {
  const stage = Math.max(0, Math.min(2, Math.trunc(step) || 0))
  const frame = guide.frames[stage]
  const Visual = visuals[guide.slug]
  return <div className={`usecase-visual${reducedMotion ? ' uv-reduced' : ''}`}>
    <div className="uv-caption" aria-hidden="true"><span>{guide.number} / {frame.label}</span><span>ILLUSTRATED EXAMPLE</span></div>
    {Visual && <div key={`${guide.slug}-${stage}`} aria-hidden="true" className="uv-stage"><Visual frame={frame} stage={stage} /></div>}
    <div className="uv-summary" role="group" aria-label="Current illustrated result">
      <strong>{frame.title}</strong><span>{frame.rows.map((row) => `${row.name}: ${row.detail}`).join(' · ')}</span><span>{frame.output}</span>
    </div>
  </div>
}
