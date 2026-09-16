import { AnimatePresence, motion, useInView, useReducedMotion } from "motion/react"
import { ArrowRight, Check, CirclePause, CirclePlay, LockKeyhole, RotateCcw, ScanSearch, Server, ShieldCheck, Terminal, Zap } from "lucide-react"
import { useEffect, useRef, useState } from "react"

const stages = [
  { label: "The interruption", short: "01 / BEFORE", heading: "Something owns :3000.", body: "Your dev server won't start. You have a port number, but no idea which process is in the way.", status: "EADDRINUSE · address already in use", action: "UNKNOWN OWNER" },
  { label: "The reveal", short: "02 / INSPECT", heading: "Now you know what it is.", body: "OpenPort shows the listener, its PID, and the rest of your local stack in one place. No process-guessing.", status: "PORT 3000 → node · PID 5678", action: "CONFLICT IDENTIFIED" },
  { label: "The resolution", short: "03 / RESOLVE", heading: "Back to building.", body: "After you confirm stopping the stale process, :3000 is free. Your database stays protected.", status: "PORT 3000 → AVAILABLE", action: "MACHINE IN ORDER" },
] as const

const services = [
  { port: "3000", name: "stale frontend", process: "node", className: "story-node-front" },
  { port: "5432", name: "database", process: "postgres", className: "story-node-db" },
  { port: "6379", name: "cache", process: "redis", className: "story-node-cache" },
  { port: "8080", name: "api", process: "java", className: "story-node-api" },
]

export default function MachineStory() {
  const reducedMotion = useReducedMotion()
  const sectionRef = useRef<HTMLElement>(null)
  const inView = useInView(sectionRef, { amount: 0.2 })
  const [stage, setStage] = useState(0)
  const [playing, setPlaying] = useState(true)
  const [cycle, setCycle] = useState(0)
  const isPlaying = playing && !reducedMotion && inView

  useEffect(() => {
    if (!isPlaying) return
    const timer = window.setInterval(() => {
      if (!document.hidden) setStage((previous) => (previous + 1) % stages.length)
    }, 4400)
    return () => window.clearInterval(timer)
  }, [isPlaying, cycle])

  function goTo(index: number) {
    setStage(index)
    setCycle((previous) => previous + 1)
  }

  return (
    <section ref={sectionRef} id="machine" className="scroll-mt-8 overflow-hidden border-b border-white/10 bg-[#0a0d0b] px-5 py-20 sm:px-8 sm:py-28 lg:px-12" aria-labelledby="machine-title">
      <div className="mx-auto max-w-[1344px] min-w-0">
        <div className="mb-12 flex min-w-0 flex-wrap items-end justify-between gap-6">
          <div className="min-w-0"><p className="eyebrow">01 / SEE THE DIFFERENCE</p><h2 id="machine-title" className="mt-5 max-w-[850px] font-display text-[clamp(2.5rem,5vw,5.4rem)] font-bold leading-[1.02] tracking-[-.065em]">A machine full of processes.<br /><span className="text-[#c8ff61]">A clear way through.</span></h2></div>
          <p className="max-w-[270px] border-l-2 border-[#c8ff61] pl-4 text-sm leading-relaxed text-[#c5cac5]">Watch one port conflict go from mystery to a deliberate fix.</p>
        </div>
        <div className="story-frame relative min-w-0 overflow-hidden rounded-2xl border border-white/15 bg-[#0d100e]">
          <div className="flex min-w-0 flex-wrap items-center justify-between gap-3 border-b border-white/10 bg-[#121613] px-5 py-4 font-mono text-[10px] uppercase tracking-[.12em] text-[#a3a8a4]"><span className="flex min-w-0 flex-wrap items-center gap-2"><span className="size-2 rounded-full bg-[#c8ff61]" /> THE OPENPORT EFFECT</span><span>ILLUSTRATED SCENARIO · NOT A LIVE SCAN</span></div>
          <div className="grid min-w-0 lg:grid-cols-[minmax(0,.42fr)_minmax(0,.58fr)]">
            <div className="flex min-w-0 flex-col border-b border-white/10 p-6 sm:p-10 lg:border-b-0 lg:border-r lg:border-white/10 lg:p-12">
              <span className="font-mono text-xs text-[#c8ff61]">{stages[stage].short}</span>
              <AnimatePresence mode="wait" initial={false}><motion.div key={stage} initial={reducedMotion ? false : { opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} exit={reducedMotion ? undefined : { opacity: 0, y: -12 }} transition={{ duration: .3 }} className="min-w-0"><h3 className="mt-7 max-w-[460px] font-display text-[clamp(1.8rem,3vw,3.3rem)] font-bold leading-[1.08] tracking-[-.055em]">{stages[stage].heading}</h3><p className="mt-5 max-w-[390px] text-sm leading-[1.75] text-[#c5cac5] sm:text-base">{stages[stage].body}</p></motion.div></AnimatePresence>
              <div className="mt-auto pt-10"><div className="flex min-w-0 flex-wrap gap-2" role="group" aria-label="Choose a scenario step">{stages.map((item, index) => <button key={item.short} type="button" onClick={() => goTo(index)} aria-current={stage === index ? "step" : undefined} className={`story-step relative min-h-12 min-w-0 flex-1 overflow-hidden rounded-lg border px-3 py-3 text-left font-mono text-[10px] leading-relaxed transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#c8ff61] active:scale-[.98] sm:text-xs ${stage === index ? "border-[#c8ff61]/60 bg-[#c8ff61]/10 text-[#f5f7f4]" : "border-white/15 bg-white/[.02] text-[#a3a8a4] hover:border-white/35 hover:text-[#f5f7f4]"}`}><span className="relative z-10">{item.label}</span>{stage === index && isPlaying && <span key={`${stage}-${cycle}`} className="story-progress absolute inset-x-0 bottom-0 h-[2px] origin-left bg-[#c8ff61]" />}</button>)}</div><div className="mt-5 flex min-w-0 flex-wrap items-center justify-between gap-3"><span className="font-mono text-[10px] text-[#a3a8a4]">{String(stage + 1).padStart(2, "0")} / 03 · {reducedMotion ? "MANUAL STEPS" : isPlaying ? "AUTO PLAYING" : playing ? "READY TO PLAY" : "PAUSED"}</span><div className="flex flex-wrap gap-2">{stage === stages.length - 1 && <button type="button" onClick={() => { goTo(0); setPlaying(true) }} className="story-control" aria-label="Replay scenario"><RotateCcw className="size-4" /> Replay</button>}{!reducedMotion && <button type="button" onClick={() => { setPlaying(!playing); setCycle((previous) => previous + 1) }} className="story-control" aria-label={playing ? "Pause animation" : "Play animation"}>{playing ? <><CirclePause className="size-4" /> Pause</> : <><CirclePlay className="size-4" /> Play</>}</button>}</div></div></div>
            </div>
            <div className="story-canvas relative flex min-w-0 flex-col p-4 sm:p-8 lg:p-10" data-stage={stage}>
              <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_38%,rgba(200,255,97,.07),transparent_60%)]" />
              <div className="relative flex min-w-0 flex-wrap items-center justify-between gap-3 border-b border-white/10 pb-4 font-mono text-[10px] uppercase tracking-[.12em] text-[#a3a8a4]"><span className="flex flex-wrap items-center gap-2"><Server className="size-4 text-[#c8ff61]" /> YOUR LOCAL MACHINE</span><span>{stage === 0 ? "? UNKNOWN" : stage === 1 ? "4 LISTENERS FOUND" : "3 LISTENERS · 1 PORT FREE"}</span></div>
              <div className="relative flex min-h-[305px] min-w-0 flex-col justify-center py-8 sm:min-h-[340px]">
                {stage === 1 && !reducedMotion && <div className="story-scan pointer-events-none absolute inset-x-0 top-0 z-10 h-px bg-[#c8ff61] shadow-[0_0_20px_3px_rgba(200,255,97,.45)]" aria-hidden="true" />}
                <div className="relative grid min-w-0 grid-cols-2 gap-3 sm:gap-4">{services.map((service, index) => <motion.div key={service.port} layout={false} initial={false} animate={reducedMotion ? undefined : { opacity: stage === 2 && index === 0 ? .32 : 1, scale: stage === 2 && index === 0 ? .96 : 1, rotate: stage === 0 ? [-2, 2, -1, 1][index] : 0, y: stage === 0 ? [4, -5, 6, -4][index] : 0 }} transition={{ duration: .55, delay: index * .06, ease: [0.22, 1, 0.36, 1] }} className={`story-node relative min-w-0 overflow-hidden rounded-xl border p-3 sm:p-4 ${service.className} ${index === 0 && stage !== 2 ? "border-[#fb7185]/45 bg-[#fb7185]/[.06]" : "border-white/15 bg-[#121613]"}`}><div className="flex min-w-0 flex-wrap items-center justify-between gap-2"><span className="font-mono text-xs font-medium text-[#f5f7f4]">:{service.port}</span><span className={`size-2 rounded-full ${index === 0 && stage !== 2 ? "bg-[#fb7185]" : stage === 2 && index === 0 ? "bg-[#c8ff61]" : "bg-[#c8ff61]"}`} /></div><div className="mt-5 min-w-0 break-all font-display text-sm font-bold sm:text-base">{stage === 0 ? service.process : index === 0 && stage === 2 ? "available" : service.name}</div><div className="mt-1 min-w-0 break-all font-mono text-[10px] text-[#a3a8a4]">{stage === 0 ? "unknown owner" : index === 0 && stage === 2 ? "ready for your app" : service.process}</div>{stage === 2 && index === 1 && <LockKeyhole className="absolute right-3 bottom-3 size-3.5 text-[#c8ff61]" aria-label="protected" />}</motion.div>)}</div>
              </div>
              <div className="relative min-w-0 rounded-xl border border-white/15 bg-[#070908] p-4 sm:p-5"><div className="flex min-w-0 flex-wrap items-start gap-3"><span className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${stage === 0 ? "bg-[#fb7185]/10 text-[#fb7185]" : "bg-[#c8ff61]/10 text-[#c8ff61]"}`}>{stage === 0 ? <Terminal className="size-4" /> : stage === 1 ? <ScanSearch className="size-4" /> : <ShieldCheck className="size-4" />}</span><div className="min-w-0 flex-1"><div className="font-mono text-[10px] uppercase tracking-[.1em] text-[#a3a8a4]">{stages[stage].action}</div><div className={`mt-1 min-w-0 break-all font-mono text-[11px] sm:text-xs ${stage === 0 ? "text-[#fb7185]" : "text-[#c8ff61]"}`}>{stages[stage].status}</div></div>{stage === 2 ? <Check className="size-4 shrink-0 text-[#c8ff61]" /> : stage === 1 ? <ArrowRight className="size-4 shrink-0 text-[#c8ff61]" /> : <Zap className="size-4 shrink-0 text-[#fb7185]" />}</div></div>
            </div>
          </div>
          <div className="flex min-w-0 flex-wrap items-center justify-between gap-2 border-t border-white/10 bg-[#121613] px-5 py-4 font-mono text-[10px] text-[#a3a8a4]"><span>INSPECT → CONFIRM → RESOLVE</span><span>Nothing on your machine is changed by this illustration.</span></div>
        </div>
      </div>
    </section>
  )
}
