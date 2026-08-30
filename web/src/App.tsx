import { AnimatePresence, motion, useInView } from "motion/react"
import {
  ArrowRight,
  Check,
  ChevronRight,
  CircleDot,
  Command,
  Copy,
  Github,
  Menu,
  Radar,
  ShieldCheck,
  Sparkles,
  X,
  Zap,
} from "lucide-react"
import { useEffect, useRef, useState } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const ports = [
  { port: "3000", process: "node", alias: "frontend", state: "LISTEN", tone: "lime" },
  { port: "5432", process: "postgres", alias: "database", state: "LISTEN", tone: "sky" },
  { port: "6379", process: "redis", alias: "cache", state: "LISTEN", tone: "violet" },
  { port: "8080", process: "java", alias: "api", state: "LISTEN", tone: "amber" },
]

const features = [
  {
    icon: Radar,
    number: "01",
    title: "See what’s running",
    text: "A clean, live view of every open port, process, and service on your machine.",
    accent: "from-lime-300/20 to-transparent",
  },
  {
    icon: Zap,
    number: "02",
    title: "Clear conflicts fast",
    text: "Find the process in your way and resolve it without stitching commands together.",
    accent: "from-sky-300/20 to-transparent",
  },
  {
    icon: ShieldCheck,
    number: "03",
    title: "Stay in control",
    text: "Protected ports and deliberate confirmations keep powerful actions predictable.",
    accent: "from-violet-300/20 to-transparent",
  },
]

function Reveal({ children, className, delay = 0 }: { children: React.ReactNode; className?: string; delay?: number }) {
  const ref = useRef<HTMLDivElement>(null)
  const inView = useInView(ref, { once: true, margin: "-80px" })
  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 24 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.65, delay, ease: [0.22, 1, 0.36, 1] }}
      className={className}
    >
      {children}
    </motion.div>
  )
}

function CopyCommand({ compact = false }: { compact?: boolean }) {
  const [copied, setCopied] = useState(false)
  const command = "go install github.com/SinghAman21/openport/cmd/openport@latest"

  const copy = async () => {
    await navigator.clipboard.writeText(command)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }

  return (
    <button
      onClick={copy}
      className={cn(
        "group flex items-center gap-3 rounded-full border border-white/10 bg-black/30 text-left font-mono text-neutral-300 transition hover:border-lime-300/40 hover:bg-black/50",
        compact ? "px-4 py-2.5 text-xs" : "w-full max-w-xl px-5 py-3.5 text-xs sm:text-sm",
      )}
      aria-label="Copy install command"
    >
      <span className="text-lime-300">$</span>
      <span className="min-w-0 flex-1 truncate">{command}</span>
      <span className="grid size-7 shrink-0 place-items-center rounded-full bg-white/[0.06] group-hover:bg-lime-300 group-hover:text-black">
        {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
      </span>
      <span className="sr-only" aria-live="polite">{copied ? "Copied" : ""}</span>
    </button>
  )
}

function Header() {
  const [open, setOpen] = useState(false)
  return (
    <header className="fixed inset-x-0 top-0 z-40 border-b border-white/[0.06] bg-ink/75 backdrop-blur-xl">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-5 lg:px-8">
        <a href="#" className="flex items-center gap-2.5 font-semibold tracking-tight">
          <span className="grid size-8 place-items-center rounded-lg border border-lime-300/20 bg-lime-300/10 font-mono text-sm text-lime-300">&gt;_</span>
          OpenPort
        </a>
        <nav className="hidden items-center gap-7 text-sm text-neutral-400 md:flex">
          <a className="transition hover:text-white" href="#features">Features</a>
          <a className="transition hover:text-white" href="#workflow">How it works</a>
          <a className="transition hover:text-white" href="https://github.com/SinghAman21/openport">GitHub</a>
        </nav>
        <div className="hidden items-center gap-2 md:flex">
          <Button asChild variant="ghost" size="sm"><a href="https://github.com/SinghAman21/openport"><Github className="size-4" />Star on GitHub</a></Button>
          <Button asChild size="sm"><a href="#install">Get OpenPort <ArrowRight className="size-3.5" /></a></Button>
        </div>
        <button className="text-neutral-300 md:hidden" onClick={() => setOpen(!open)} aria-label="Toggle menu">
          {open ? <X /> : <Menu />}
        </button>
      </div>
      <AnimatePresence>
        {open && (
          <motion.nav initial={{ height: 0, opacity: 0 }} animate={{ height: "auto", opacity: 1 }} exit={{ height: 0, opacity: 0 }} className="overflow-hidden border-t border-white/[0.06] bg-ink md:hidden">
            <div className="flex flex-col gap-4 px-5 py-5 text-sm text-neutral-300">
              <a href="#features" onClick={() => setOpen(false)}>Features</a>
              <a href="#workflow" onClick={() => setOpen(false)}>How it works</a>
              <a href="https://github.com/SinghAman21/openport">GitHub</a>
              <Button asChild><a href="#install" onClick={() => setOpen(false)}>Get OpenPort</a></Button>
            </div>
          </motion.nav>
        )}
      </AnimatePresence>
    </header>
  )
}

function TerminalDemo() {
  const [selected, setSelected] = useState(0)
  useEffect(() => {
    const timer = window.setInterval(() => setSelected((value) => (value + 1) % ports.length), 2200)
    return () => window.clearInterval(timer)
  }, [])

  return (
    <motion.div
      initial={{ opacity: 0, y: 35, scale: 0.97 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.9, delay: 0.25, ease: [0.22, 1, 0.36, 1] }}
      className="terminal-shadow relative mx-auto w-full max-w-5xl overflow-hidden rounded-2xl border border-white/10 bg-[#0a0d0b] text-left"
    >
      <div className="flex h-11 items-center gap-2 border-b border-white/[0.07] bg-white/[0.025] px-4">
        <span className="size-2.5 rounded-full bg-[#ff6159]" /><span className="size-2.5 rounded-full bg-[#ffbd2e]" /><span className="size-2.5 rounded-full bg-[#28c941]" />
        <span className="absolute left-1/2 -translate-x-1/2 font-mono text-[10px] text-neutral-600 sm:text-xs">openport — ports</span>
      </div>
      <div className="grid min-h-[330px] md:grid-cols-[1fr_280px]">
        <div className="p-4 sm:p-6">
          <div className="mb-5 flex items-center justify-between">
            <div className="flex items-center gap-2 font-mono text-xs"><span className="size-2 animate-pulse rounded-full bg-lime-300" /><span className="text-neutral-300">4 services running</span></div>
            <span className="font-mono text-[10px] text-neutral-600">LIVE</span>
          </div>
          <div className="mb-2 grid grid-cols-[70px_1fr_1fr] gap-3 px-3 font-mono text-[10px] uppercase tracking-[.18em] text-neutral-600 sm:grid-cols-[80px_1fr_1fr_80px]">
            <span>Port</span><span>Process</span><span>Alias</span><span className="hidden sm:block">State</span>
          </div>
          <div className="space-y-1">
            {ports.map((item, index) => (
              <motion.button
                key={item.port}
                onClick={() => setSelected(index)}
                animate={{ backgroundColor: selected === index ? "rgba(200,255,97,.08)" : "rgba(255,255,255,0)" }}
                className={cn("relative grid w-full grid-cols-[70px_1fr_1fr] gap-3 rounded-lg px-3 py-3 text-left font-mono text-xs transition sm:grid-cols-[80px_1fr_1fr_80px]", selected === index ? "text-white" : "text-neutral-500 hover:bg-white/[.03]")}
              >
                {selected === index && <motion.span layoutId="active-port" className="absolute inset-y-2 left-0 w-0.5 rounded-full bg-lime-300" />}
                <span className="text-lime-300">:{item.port}</span><span>{item.process}</span><span>{item.alias}</span><span className="hidden text-[10px] text-neutral-500 sm:block">{item.state}</span>
              </motion.button>
            ))}
          </div>
        </div>
        <div className="border-t border-white/[0.07] bg-white/[0.018] p-6 md:border-t-0 md:border-l">
          <AnimatePresence mode="wait">
            <motion.div key={selected} initial={{ opacity: 0, x: 8 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -8 }} transition={{ duration: .2 }}>
              <p className="mb-2 font-mono text-[10px] uppercase tracking-[.18em] text-neutral-600">Selected service</p>
              <p className="mb-6 text-xl font-semibold text-white">{ports[selected].alias}</p>
              <dl className="space-y-3 font-mono text-xs">
                <div className="flex justify-between"><dt className="text-neutral-600">Port</dt><dd className="text-neutral-300">{ports[selected].port}</dd></div>
                <div className="flex justify-between"><dt className="text-neutral-600">Process</dt><dd className="text-neutral-300">{ports[selected].process}</dd></div>
                <div className="flex justify-between"><dt className="text-neutral-600">Health</dt><dd className="flex items-center gap-1.5 text-lime-300"><CircleDot className="size-3" />Healthy</dd></div>
              </dl>
              <button className="mt-8 flex w-full items-center justify-between rounded-lg border border-white/10 bg-white/[.03] px-3 py-2.5 font-mono text-xs text-neutral-300 transition hover:border-lime-300/30 hover:text-white"><span>View details</span><ChevronRight className="size-3.5" /></button>
            </motion.div>
          </AnimatePresence>
        </div>
      </div>
      <div className="flex items-center justify-between border-t border-white/[0.07] px-5 py-3 font-mono text-[10px] text-neutral-600"><span>↑↓ navigate · enter inspect · x stop</span><span>esc back</span></div>
    </motion.div>
  )
}

function App() {
  return (
    <div className="noise min-h-screen bg-ink text-white">
      <Header />
      <main>
        <section className="relative overflow-hidden px-5 pb-24 pt-36 sm:pt-44 lg:px-8">
          <div className="grid-bg absolute inset-0" />
          <div className="absolute left-1/2 top-10 h-[520px] w-[720px] -translate-x-1/2 rounded-full bg-lime-300/[0.07] blur-[130px]" />
          <div className="relative mx-auto max-w-7xl text-center">
            <motion.a initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} href="https://github.com/SinghAman21/openport" className="mb-7 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[.04] px-3.5 py-2 text-xs text-neutral-300 backdrop-blur transition hover:border-lime-300/30">
              <Sparkles className="size-3.5 text-lime-300" /><span>Open source. Built for your local stack.</span><ChevronRight className="size-3.5 text-neutral-600" />
            </motion.a>
            <motion.h1 initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: .08, duration: .7 }} className="text-balance mx-auto max-w-5xl text-5xl font-semibold leading-[.98] tracking-[-.055em] sm:text-7xl lg:text-[96px]">
              Your ports.<br /><span className="text-neutral-500">Under control.</span>
            </motion.h1>
            <motion.p initial={{ opacity: 0, y: 15 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: .16, duration: .7 }} className="text-balance mx-auto mt-7 max-w-2xl text-base leading-relaxed text-neutral-400 sm:text-lg">
              See what’s running, understand what owns it, and resolve conflicts in seconds—all from one focused interface.
            </motion.p>
            <motion.div initial={{ opacity: 0, y: 15 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: .24, duration: .7 }} className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Button asChild size="lg"><a href="#install">Install OpenPort <ArrowRight className="size-4" /></a></Button>
              <Button asChild variant="outline" size="lg"><a href="https://github.com/SinghAman21/openport"><Github className="size-4" />View on GitHub</a></Button>
            </motion.div>
            <p className="mt-4 font-mono text-[10px] uppercase tracking-[.2em] text-neutral-600">Free · Offline-first · macOS, Linux & Windows</p>
            <div className="mt-16 sm:mt-20"><TerminalDemo /></div>
          </div>
        </section>

        <section className="overflow-hidden border-y border-white/[0.06] py-5">
          <div className="marquee flex w-max items-center">
            {[0, 1].map((set) => (
              <div key={set} className="flex items-center gap-10 pr-10 font-mono text-xs uppercase tracking-[.2em] text-neutral-600">
                <span>One command</span><span className="text-lime-300">✦</span><span>Every port</span><span className="text-lime-300">✦</span><span>Zero guesswork</span><span className="text-lime-300">✦</span><span>Local first</span><span className="text-lime-300">✦</span><span>Built for flow</span><span className="text-lime-300">✦</span>
              </div>
            ))}
          </div>
        </section>

        <section id="features" className="px-5 py-24 sm:py-32 lg:px-8">
          <div className="mx-auto max-w-7xl">
            <Reveal className="mb-14 max-w-2xl">
              <p className="mb-4 font-mono text-xs uppercase tracking-[.2em] text-lime-300">The simple way through</p>
              <h2 className="text-balance text-4xl font-semibold tracking-[-.04em] sm:text-6xl">Less hunting.<br /><span className="text-neutral-500">More building.</span></h2>
            </Reveal>
            <div className="grid gap-4 lg:grid-cols-3">
              {features.map((feature, index) => (
                <Reveal key={feature.title} delay={index * .1}>
                  <article className="group relative min-h-[330px] overflow-hidden rounded-2xl border border-white/[0.08] bg-panel p-7 transition duration-500 hover:-translate-y-1 hover:border-white/[0.16]">
                    <div className={cn("absolute inset-0 bg-gradient-to-br opacity-0 transition duration-500 group-hover:opacity-100", feature.accent)} />
                    <div className="relative flex h-full min-h-[274px] flex-col">
                      <div className="flex items-start justify-between"><span className="grid size-11 place-items-center rounded-xl border border-white/10 bg-white/[.04]"><feature.icon className="size-5 text-lime-300" /></span><span className="font-mono text-xs text-neutral-700">{feature.number}</span></div>
                      <div className="mt-auto"><h3 className="mb-3 text-2xl font-semibold tracking-tight">{feature.title}</h3><p className="max-w-sm leading-relaxed text-neutral-500">{feature.text}</p></div>
                    </div>
                  </article>
                </Reveal>
              ))}
            </div>
          </div>
        </section>

        <section id="workflow" className="px-5 pb-24 sm:pb-32 lg:px-8">
          <div className="mx-auto grid max-w-7xl overflow-hidden rounded-3xl border border-white/[0.08] bg-[#0b0e0c] lg:grid-cols-2">
            <Reveal className="flex flex-col justify-center p-8 sm:p-14">
              <span className="mb-8 grid size-12 place-items-center rounded-xl bg-lime-300 text-black"><Command className="size-5" /></span>
              <h2 className="text-balance text-4xl font-semibold tracking-[-.04em] sm:text-5xl">From “what’s using that port?” to done.</h2>
              <p className="mt-5 max-w-lg leading-relaxed text-neutral-400">Launch once. Everything you need is visible and reachable—no command archaeology required.</p>
              <ul className="mt-8 space-y-4 text-sm text-neutral-300">
                {["Live process and port visibility", "Aliases that make services recognizable", "Safe, confirmed actions when conflicts happen"].map((item) => <li key={item} className="flex items-center gap-3"><span className="grid size-5 place-items-center rounded-full bg-lime-300/10 text-lime-300"><Check className="size-3" /></span>{item}</li>)}
              </ul>
            </Reveal>
            <div className="relative min-h-[440px] overflow-hidden border-t border-white/[0.07] bg-black/20 p-6 lg:border-l lg:border-t-0 sm:p-10">
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_70%_30%,rgba(200,255,97,.12),transparent_45%)]" />
              <Reveal className="relative h-full">
                <div className="absolute left-0 top-8 w-[92%] rounded-xl border border-white/10 bg-[#101411] p-5 shadow-2xl sm:left-8">
                  <div className="mb-5 flex items-center justify-between"><span className="font-mono text-xs text-neutral-500">COMMAND PALETTE</span><span className="rounded border border-white/10 px-1.5 py-0.5 font-mono text-[9px] text-neutral-600">ESC</span></div>
                  <div className="flex items-center gap-3 rounded-lg border border-lime-300/20 bg-lime-300/[.04] px-3 py-2.5"><span className="text-lime-300">›</span><span className="cursor font-mono text-sm text-neutral-200">show port 3000</span></div>
                  <div className="mt-3 space-y-1 font-mono text-xs text-neutral-500"><div className="rounded-lg bg-white/[.04] px-3 py-2.5 text-white">Inspect port 3000</div><div className="px-3 py-2.5">Check health</div><div className="px-3 py-2.5">Create alias</div></div>
                </div>
                <motion.div animate={{ y: [0, -6, 0] }} transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }} className="absolute bottom-7 right-0 w-[78%] rounded-xl border border-white/10 bg-[#121613] p-5 shadow-2xl sm:right-5">
                  <div className="mb-4 flex items-center gap-2"><span className="size-2 rounded-full bg-lime-300" /><span className="font-mono text-xs text-neutral-300">Port 3000 is clear</span></div><p className="text-sm text-neutral-500">The previous process was stopped safely.</p>
                </motion.div>
              </Reveal>
            </div>
          </div>
        </section>

        <section id="install" className="px-5 pb-16 lg:px-8">
          <Reveal className="mx-auto max-w-7xl overflow-hidden rounded-3xl border border-lime-300/20 bg-lime-300 p-8 text-neutral-950 sm:p-14 lg:p-20">
            <div className="flex flex-col items-start justify-between gap-10 lg:flex-row lg:items-end">
              <div><p className="mb-5 font-mono text-xs font-semibold uppercase tracking-[.2em]">Ready when you are</p><h2 className="text-balance max-w-3xl text-4xl font-semibold tracking-[-.05em] sm:text-6xl lg:text-7xl">Make port conflicts a five-second problem.</h2></div>
              <Button asChild size="lg" className="shrink-0 bg-neutral-950 text-white shadow-xl hover:bg-neutral-800"><a href="https://github.com/SinghAman21/openport">Get OpenPort <ArrowRight className="size-4" /></a></Button>
            </div>
            <div className="mt-12 border-t border-black/15 pt-7"><CopyCommand /></div>
          </Reveal>
        </section>
      </main>
      <footer className="px-5 py-10 lg:px-8"><div className="mx-auto flex max-w-7xl flex-col items-start justify-between gap-5 border-t border-white/[0.06] pt-8 text-sm text-neutral-600 sm:flex-row sm:items-center"><div className="flex items-center gap-2 text-neutral-400"><span className="font-mono text-lime-300">&gt;_</span><span className="font-semibold">OpenPort</span></div><p>Built for developers who have better things to do.</p><a className="flex items-center gap-2 transition hover:text-white" href="https://github.com/SinghAman21/openport"><Github className="size-4" />GitHub</a></div></footer>
    </div>
  )
}

export default App
