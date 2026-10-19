import { useInView, useReducedMotion } from "motion/react"
import { ArrowLeft, ArrowRight, Check, ChevronRight, CirclePause, CirclePlay, Copy, Github, RotateCcw, X } from "lucide-react"
import { useEffect, useRef, useState } from "react"

import UseCaseVisual from "@/components/UseCaseVisual"
import { guides, type Guide } from "@/lib/usecases"

const repo = "https://github.com/SinghAman21/openport"
const duration = 5600

function CommandBlock({ value, caption }: { value: string; caption: string }) {
  const [status, setStatus] = useState<"idle" | "copied" | "error">("idle")
  useEffect(() => {
    if (status === "idle") return
    const timer = window.setTimeout(() => setStatus("idle"), 3000)
    return () => window.clearTimeout(timer)
  }, [status])
  async function copy() {
    try { await navigator.clipboard.writeText(value); setStatus("copied") }
    catch { setStatus("error") }
  }
  return <div className="min-w-0"><p className="mb-3 font-mono text-[11px] uppercase tracking-[.15em] text-[#a3a8a4]">TRY THIS / YOUR TERMINAL</p><div className="flex min-w-0 flex-wrap items-center gap-3 rounded-xl border border-[#c8ff61]/30 bg-[#121613] p-4 sm:p-5"><span className="font-mono text-[#c8ff61]">$</span><code className="min-w-0 flex-1 break-all font-mono text-xs text-[#f5f7f4] sm:text-sm">{value}</code><button type="button" onClick={copy} className="story-control shrink-0" aria-label={`Copy command ${value}`}>{status === "copied" ? <><Check className="size-4" /> Copied</> : <><Copy className="size-4" /> Copy</>}</button></div><p className="mt-3 text-sm text-[#c5cac5]">{caption}</p><p className="min-h-5 text-xs text-[#c8ff61]" role="status" aria-live="polite">{status === "error" ? "Clipboard unavailable. Select the command above to copy it manually." : status === "copied" ? "Command copied." : ""}</p></div>
}

function Workflow({ guide }: { guide: Guide }) {
  const ref = useRef<HTMLDivElement>(null)
  const visible = useInView(ref, { amount: .2 })
  const reducedMotion = useReducedMotion()
  const [step, setStep] = useState(0)
  const [playing, setPlaying] = useState(true)
  const [progress, setProgress] = useState(0)
  const elapsed = useRef(0)
  const lastTick = useRef(0)
  const running = playing && visible && !reducedMotion
  useEffect(() => {
    if (!running) return
    lastTick.current = performance.now()
    const timer = window.setInterval(() => {
      const now = performance.now()
      if (document.hidden) { lastTick.current = now; return }
      elapsed.current += now - lastTick.current
      lastTick.current = now
      if (elapsed.current >= duration) {
        elapsed.current %= duration
        setStep((previous) => (previous + 1) % guide.frames.length)
      }
      setProgress((elapsed.current / duration) * 100)
    }, 80)
    return () => {
      window.clearInterval(timer)
      if (playing && visible && !reducedMotion) elapsed.current += performance.now() - lastTick.current
    }
  }, [running, guide.frames.length, playing, visible, reducedMotion])
  function select(index: number) {
    elapsed.current = 0
    lastTick.current = performance.now()
    setProgress(0)
    setStep(index)
  }
  const frame = guide.frames[step]
  return <div ref={ref} className="min-w-0 overflow-hidden rounded-2xl border border-white/15 bg-[#0d100e]" aria-label={`Illustrated ${guide.category.toLowerCase()} workflow`}>
    <div className="flex min-w-0 flex-wrap items-center justify-between gap-2 border-b border-white/10 bg-[#121613] px-5 py-4 font-mono text-[10px] uppercase tracking-[.13em] text-[#a3a8a4]"><span className="flex items-center gap-2"><span className="size-2 rounded-full bg-[#c8ff61]" /> A SCENE FROM YOUR MACHINE</span><span>ILLUSTRATION · NOT A LIVE SCAN</span></div>
    <div className="grid min-w-0 lg:grid-cols-[minmax(0,.42fr)_minmax(0,.58fr)]"><div className="flex min-w-0 flex-col border-b border-white/10 p-5 sm:p-9 lg:border-b-0 lg:border-r lg:border-white/10 lg:p-11"><p className="eyebrow">SCENE {String(step + 1).padStart(2, "0")} / 03 · {frame.label}</p><h2 className="mt-6 min-w-0 break-words font-display text-3xl font-bold leading-[1.1] tracking-[-.055em] sm:text-4xl">{frame.title}</h2><p className="mt-6 border-l-2 border-[#c8ff61] pl-4 font-display text-base font-medium leading-[1.7] text-[#f5f7f4] sm:text-lg">{guide.steps[step].story}</p><div className="mt-auto pt-9"><div className="flex min-w-0 flex-wrap gap-2" role="group" aria-label="Select a story scene">{guide.frames.map((item, index) => <button key={item.label} type="button" onClick={() => select(index)} aria-current={step === index ? "step" : undefined} className={`relative min-h-12 min-w-0 flex-1 overflow-hidden rounded-lg border px-3 py-3 text-left font-mono text-[10px] leading-relaxed transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#c8ff61] active:scale-[.98] ${step === index ? "border-[#c8ff61]/60 bg-[#c8ff61]/10 text-[#f5f7f4]" : "border-white/15 text-[#a3a8a4] hover:border-white/35 hover:text-[#f5f7f4]"}`}><span className="relative z-10">{String(index + 1).padStart(2, "0")} / {item.label}</span>{step === index && <span className="pointer-events-none absolute inset-y-0 left-0 bg-[#c8ff61]/10" style={{ width: `${progress}%` }} aria-hidden="true" />}</button>)}</div><div className="mt-4 flex min-w-0 flex-wrap items-center justify-between gap-3"><span className="font-mono text-[10px] text-[#a3a8a4]">{reducedMotion ? "MANUAL MODE" : running ? "PLAYING" : playing ? "AUTO ON · STARTS IN VIEW" : "PAUSED"}</span><div className="flex flex-wrap gap-2"><button type="button" className="story-control" onClick={() => { select(0); setPlaying(true) }} aria-label="Replay story"><RotateCcw className="size-4" /> Replay</button>{!reducedMotion && <button type="button" className="story-control" onClick={() => setPlaying((current) => !current)} aria-label={playing ? visible ? "Pause story" : "Turn off autoplay" : "Resume story"}>{playing ? <><CirclePause className="size-4" /> Pause</> : <><CirclePlay className="size-4" /> Play</>}</button>}</div></div></div></div><UseCaseVisual guide={guide} step={step} reducedMotion={!!reducedMotion} /></div>
    <div className="flex min-w-0 flex-wrap justify-between gap-2 border-t border-white/10 bg-[#121613] px-5 py-3 font-mono text-[10px] uppercase tracking-[.1em] text-[#a3a8a4]"><span>CHOOSE EACH SCENE OR LET THE STORY PLAY</span><span>STEP {step + 1} / 3</span></div>
  </div>
}

function Article({ guide }: { guide: Guide }) {
  useEffect(() => { document.title = `${guide.title} — OpenPort Use Cases` }, [guide])
  const next = guides[(guides.indexOf(guide) + 1) % guides.length]
  return <div className="site min-h-screen overflow-x-hidden bg-[#070908] text-[#f5f7f4]"><header className="border-b border-white/10 bg-[#070908]"><div className="mx-auto flex min-h-[76px] max-w-[1344px] flex-wrap items-center justify-between gap-3 px-5 sm:px-8 lg:px-12"><a href="/" className="inline-flex min-h-11 flex-wrap items-center gap-2 font-display text-lg font-bold tracking-[-.055em] text-[#f5f7f4] hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]"><span className="text-[#c8ff61]">↗</span> openport <span className="font-mono text-[10px] font-normal tracking-normal text-[#a3a8a4]">/ FIELD NOTES</span></a><a href="/usecase" className="nav-link gap-2"><ArrowLeft className="size-4" /> All use cases</a></div></header><main><article>
    <div className="hero-grid border-b border-white/10 px-5 pb-16 pt-10 sm:px-8 sm:pb-20 sm:pt-14 lg:px-12"><div className="mx-auto max-w-[1344px] min-w-0"><nav aria-label="Breadcrumb" className="mb-12 flex min-w-0 flex-wrap items-center gap-2 font-mono text-[11px] text-[#a3a8a4]"><a className="inline-flex min-h-11 items-center hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]" href="/">Home</a><ChevronRight className="size-3" /><a className="inline-flex min-h-11 items-center hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]" href="/usecase">Use cases</a><ChevronRight className="size-3" /><span className="text-[#c8ff61]">{guide.category.toLowerCase()}</span></nav><div className="grid min-w-0 items-end gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,.42fr)]"><div className="min-w-0"><p className="eyebrow">FIELD NOTE / {guide.number} · {guide.category} · {guide.time}</p><h1 className="mt-6 max-w-[900px] min-w-0 break-words font-display text-[clamp(3rem,6.4vw,7rem)] font-bold leading-[.99] tracking-[-.07em]">{guide.title}</h1></div><div className="min-w-0 border-l-2 border-[#c8ff61] pl-5"><p className="font-display text-xl font-semibold leading-snug text-[#c8ff61] sm:text-2xl">{guide.promise}</p><p className="mt-4 text-sm leading-[1.8] text-[#c5cac5]">{guide.lead}</p></div></div></div></div>
    <section className="border-b border-white/10 px-5 py-16 sm:px-8 sm:py-24 lg:px-12" aria-label="The problem and two approaches"><div className="mx-auto max-w-[1344px] min-w-0"><div className="grid min-w-0 gap-5 border-b border-white/15 pb-10 lg:grid-cols-[minmax(0,.32fr)_minmax(0,.68fr)] lg:gap-12"><div><p className="eyebrow">01 / THE PROBLEM</p><h2 className="mt-4 font-display text-3xl font-bold tracking-[-.055em] sm:text-4xl">Sound familiar?</h2></div><p className="min-w-0 max-w-2xl text-lg leading-[1.75] text-[#f5f7f4] sm:text-xl">{guide.problem}</p></div><div className="grid min-w-0 gap-0 lg:grid-cols-2"><div className="min-w-0 border-b border-white/15 py-9 lg:border-b-0 lg:border-r lg:pr-12"><p className="font-mono text-[11px] uppercase tracking-[.14em] text-[#fb7185]">02 / THE OLD WAY</p><h3 className="mt-4 font-display text-2xl font-bold tracking-[-.045em]">{guide.oldApproach.title}</h3><p className="mt-3 max-w-md text-sm leading-[1.75] text-[#c5cac5]">{guide.oldApproach.body}</p><code className="mt-6 block min-w-0 max-w-full break-all rounded-lg border border-white/10 bg-[#0d100e] px-4 py-4 font-mono text-xs text-[#f5f7f4]">$ {guide.oldApproach.command}</code></div><div className="min-w-0 py-9 lg:pl-12"><p className="eyebrow">03 / WITH OPENPORT</p><h3 className="mt-4 font-display text-2xl font-bold tracking-[-.045em]">{guide.newApproach.title}</h3><p className="mt-3 max-w-md text-sm leading-[1.75] text-[#c5cac5]">{guide.newApproach.body}</p><p className="mt-6 min-w-0 break-all border-l-2 border-[#c8ff61] py-2 pl-4 font-mono text-xs text-[#c8ff61]">$ {guide.command}</p></div></div></div></section>
    <section className="px-5 py-16 sm:px-8 sm:py-24 lg:px-12" aria-label="Animated story"><div className="mx-auto max-w-[1344px] min-w-0"><div className="mb-6 flex min-w-0 flex-wrap items-end justify-between gap-5"><div><p className="eyebrow">THE STORY / IN THREE SCENES</p><h2 className="mt-3 font-display text-3xl font-bold tracking-[-.05em] sm:text-4xl">See the difference.</h2></div><p className="max-w-[260px] text-sm leading-relaxed text-[#a3a8a4]">Illustrated with example data. Nothing here changes your machine.</p></div><Workflow guide={guide} /></div></section>
    <section className="border-y border-white/10 bg-[#0a0d0b] px-5 py-16 sm:px-8 sm:py-24 lg:px-12"><div className="mx-auto grid max-w-[1344px] min-w-0 gap-12 lg:grid-cols-[minmax(0,.45fr)_minmax(0,.55fr)] lg:gap-20"><div className="min-w-0"><p className="eyebrow">THE PRACTICAL PART</p><h2 className="mt-5 font-display text-[clamp(2.4rem,4vw,4.5rem)] font-bold leading-[1.04] tracking-[-.06em]">Try it on<br /><span className="text-[#c8ff61]">your machine.</span></h2><p className="mt-6 max-w-[360px] text-base leading-[1.8] text-[#c5cac5]">The story uses example data. Here’s how to put the same idea to work in your own terminal.</p><div className="mt-8"><CommandBlock value={guide.command} caption={guide.commandNote} /></div></div><ol className="min-w-0 border-t border-white/15">{guide.steps.map((item, index) => <li key={item.title} className="grid min-w-0 grid-cols-[34px_minmax(0,1fr)] gap-4 border-b border-white/15 py-7 sm:grid-cols-[56px_minmax(0,1fr)] sm:gap-6"><span className="pt-1 font-mono text-xs text-[#c8ff61]">0{index + 1} /</span><div className="min-w-0"><h3 className="break-words font-display text-xl font-bold tracking-[-.04em] sm:text-2xl">{item.title}</h3><p className="mt-3 max-w-lg break-words text-sm leading-[1.8] text-[#c5cac5]">{item.body}</p></div></li>)}</ol></div></section>
    <section className="px-5 py-16 sm:px-8 sm:py-24 lg:px-12"><div className="mx-auto grid max-w-[1344px] min-w-0 gap-10 rounded-2xl border border-[#c8ff61]/30 bg-[#121613] p-7 sm:p-12 lg:grid-cols-[minmax(0,1fr)_minmax(0,.7fr)] lg:items-end"><div className="min-w-0"><p className="eyebrow">THE PAYOFF</p><h2 className="mt-4 max-w-[700px] break-words font-display text-[clamp(2rem,3.7vw,4rem)] font-bold leading-[1.08] tracking-[-.055em]">Less friction.<br /><span className="text-[#c8ff61]">More time for the actual work.</span></h2><p className="mt-5 max-w-[650px] leading-[1.8] text-[#c5cac5]">{guide.outcome}</p></div><div className="flex min-w-0 flex-wrap gap-3 lg:justify-end"><a href="/#install" className="button-lime min-h-12">Get OpenPort <ArrowRight className="size-4" /></a><a href={repo} target="_blank" rel="noopener noreferrer" className="button-outline min-h-12"><Github className="size-4" /> Source & docs</a></div></div><div className="mx-auto mt-10 flex max-w-[1344px] min-w-0 flex-wrap items-center justify-between gap-4 border-t border-white/10 pt-8"><a href="/usecase" className="nav-link gap-2"><ArrowLeft className="size-4" /> All field notes</a><a href={`/usecase/${next.slug}`} className="inline-flex min-h-11 min-w-0 flex-wrap items-center gap-3 text-right text-sm font-semibold text-[#c8ff61] hover:text-[#d7ff8a] focus-visible:outline-2 focus-visible:outline-[#c8ff61]">Next: {next.title} <ArrowRight className="size-4 shrink-0" /></a></div></section>
    </article></main><footer className="border-t border-white/10 px-5 py-6 sm:px-8 lg:px-12"><div className="mx-auto flex max-w-[1344px] flex-wrap items-center justify-between gap-3 text-xs text-[#a3a8a4]"><span>openport / LOCAL FIRST</span><span>Illustrations, not live scans.</span></div></footer></div>
}

export default function UseCasePage({ slug }: { slug: string }) {
  const guide = guides.find((item) => item.slug === slug)
  useEffect(() => { if (!guide) document.title = "Use case not found — OpenPort" }, [guide])
  if (!guide) return <main className="flex min-h-screen flex-col items-start justify-center gap-5 bg-[#070908] px-6 text-[#f5f7f4]"><X className="size-8 text-[#fb7185]" /><h1 className="font-display text-4xl font-bold">No field note at this address.</h1><p className="text-[#c5cac5]">The guide may have moved. Pick one from the use cases index.</p><a className="button-lime" href="/usecase">See all use cases <ArrowRight className="size-4" /></a></main>
  return <Article guide={guide} />
}
