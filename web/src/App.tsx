import { AnimatePresence, motion } from "motion/react"
import {
  ArrowDownRight, ArrowRight, Check, CheckCheck, ChevronRight, CircleHelp,
  Copy, ExternalLink, Github, LockKeyhole, Menu, Search,
  Shield, ShieldCheck, Terminal, X, Zap,
} from "lucide-react"
import { useEffect, useState } from "react"

const repo = "https://github.com/SinghAman21/openport"
const command = "go install github.com/SinghAman21/openport/cmd/openport@latest"

const initialPorts = [
  { port: "3000", name: "frontend", process: "node", pid: "5678", address: "127.0.0.1", health: "Healthy", protected: false },
  { port: "5432", name: "database", process: "postgres", pid: "1234", address: "127.0.0.1", health: "Healthy", protected: true },
  { port: "6379", name: "cache", process: "redis-server", pid: "9012", address: "127.0.0.1", health: "Unchecked", protected: false },
  { port: "8080", name: "api", process: "java", pid: "4821", address: "0.0.0.0", health: "Healthy", protected: false },
]

type Port = (typeof initialPorts)[number]

function Mark() {
  return <span className="brand-mark" aria-hidden="true"><span className="brand-mark-dot" /><span className="brand-mark-line" /></span>
}

function CopyInstall({ light = false }: { light?: boolean }) {
  const [status, setStatus] = useState<"idle" | "copied" | "error">("idle")
  async function copy() {
    try {
      await navigator.clipboard.writeText(command)
      setStatus("copied")
    } catch {
      setStatus("error")
    }
  }
  useEffect(() => {
    if (status === "idle") return
    const timer = window.setTimeout(() => setStatus("idle"), 3500)
    return () => window.clearTimeout(timer)
  }, [status])
  return (
    <div className="min-w-0 max-w-full">
      <button type="button" onClick={copy} className={`copy-command group flex min-h-14 w-full min-w-0 max-w-full flex-wrap items-center gap-3 rounded-xl border px-4 py-3 text-left transition duration-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#c8ff61] active:scale-[.99] sm:px-5 ${light ? "border-[#070908]/20 bg-[#070908] text-[#f5f7f4] hover:bg-[#202820]" : "border-white/15 bg-[#121613] text-[#f5f7f4] hover:border-[#c8ff61]/50 hover:bg-[#1b211b]"}`} aria-label="Copy Go install command">
        <span className="shrink-0 font-mono text-[#c8ff61]">$</span>
        <span className="min-w-0 flex-1 break-all font-mono text-[11px] leading-relaxed sm:text-sm">{command}</span>
        <span className="ml-auto flex min-h-9 shrink-0 flex-wrap items-center gap-2 font-mono text-xs text-[#c8ff61]">{status === "copied" ? <><Check className="size-4" />Copied</> : status === "error" ? "Copy failed" : <><Copy className="size-4" />Copy</>}</span>
      </button>
      <span className={`mt-2 block min-h-5 text-xs ${light ? "text-[#26331b]" : "text-[#a3a8a4]"}`} role="status" aria-live="polite">{status === "error" ? "Clipboard access was blocked. Select the command above to copy it manually." : status === "copied" ? "Install command copied to clipboard." : ""}</span>
    </div>
  )
}

function Header() {
  const [open, setOpen] = useState(false)
  useEffect(() => {
    if (!open) return
    function close(event: KeyboardEvent) { if (event.key === "Escape") setOpen(false) }
    window.addEventListener("keydown", close)
    return () => window.removeEventListener("keydown", close)
  }, [open])
  return (
    <header className="relative z-20 border-b border-white/10 bg-[#070908]">
      <div className="mx-auto flex min-h-[76px] max-w-[1440px] flex-wrap items-center justify-between gap-3 px-5 sm:px-8 lg:px-12">
        <a href="#top" className="flex min-h-11 flex-wrap items-center gap-3 rounded-sm font-display text-[18px] font-bold tracking-[-.06em] text-[#f5f7f4] transition hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]" onClick={() => setOpen(false)}><Mark />openport<span className="self-start pt-2 font-mono text-[9px] tracking-normal text-[#c8ff61]">/ CLI</span></a>
        <nav aria-label="Main navigation" className="hidden flex-wrap items-center gap-8 md:flex">
          <a className="nav-link" href="#features">Features</a>
          <a className="nav-link" href="#how-it-works">How it works</a>
          <a className="nav-link" href="#install">Install</a>
        </nav>
        <div className="hidden flex-wrap items-center gap-4 md:flex"><a className="nav-link flex items-center gap-2" href={repo} target="_blank" rel="noopener noreferrer"><Github className="size-4" /> GitHub <ExternalLink className="size-3" /></a><a href="#install" className="button-lime">Get OpenPort <ArrowRight className="size-4" /></a></div>
        <button type="button" className="flex size-11 items-center justify-center rounded-lg border border-white/15 text-[#f5f7f4] transition hover:border-[#c8ff61] hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61] active:bg-white/10 md:hidden" aria-label={open ? "Close menu" : "Open menu"} aria-controls="mobile-nav" aria-expanded={open} onClick={() => setOpen(!open)}>{open ? <X className="size-5" /> : <Menu className="size-5" />}</button>
      </div>
      {open && <nav id="mobile-nav" aria-label="Mobile navigation" className="flex flex-col border-t border-white/10 bg-[#0d100e] px-5 py-3 md:hidden">{[["Features", "#features"], ["How it works", "#how-it-works"], ["Install", "#install"], ["GitHub ↗", repo]].map(([label, href]) => <a key={label} className="flex min-h-12 items-center border-b border-white/10 text-sm text-[#f5f7f4] hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]" href={href} onClick={() => setOpen(false)}>{label}</a>)}</nav>}
    </header>
  )
}

function ConsolePreview() {
  const [ports, setPorts] = useState<Port[]>(initialPorts)
  const [selected, setSelected] = useState("3000")
  const [query, setQuery] = useState("")
  const [checking, setChecking] = useState(false)
  const [notice, setNotice] = useState("")
  const current = ports.find((item) => item.port === selected) ?? ports[0]
  const filtered = ports.filter((item) => `${item.port} ${item.name} ${item.process}`.toLowerCase().includes(query.toLowerCase()))
  useEffect(() => {
    if (!checking) return
    const timer = window.setTimeout(() => {
      setPorts((items) => items.map((item) => item.port === selected ? { ...item, health: "Healthy" } : item))
      setChecking(false)
      setNotice(`Port ${selected} is responding in this preview.`)
    }, 700)
    return () => window.clearTimeout(timer)
  }, [checking, selected])
  function selectPort(port: string) { setSelected(port); setChecking(false); setNotice("") }
  function toggleProtection() {
    const next = !current.protected
    setPorts((items) => items.map((item) => item.port === current.port ? { ...item, protected: next } : item))
    setNotice(`Port ${current.port} ${next ? "protected" : "unprotected"} in this preview.`)
  }
  return (
    <div className="console relative min-w-0 overflow-hidden rounded-2xl border border-white/[.14] bg-[#0a0d0b] shadow-[0_24px_100px_rgba(0,0,0,.45)]" aria-label="Interactive OpenPort interface preview">
      <div className="flex min-h-12 flex-wrap items-center justify-between gap-2 border-b border-white/10 bg-[#121613] px-4 sm:px-5"><div className="flex flex-wrap items-center gap-2"><span className="size-2 rounded-full bg-[#fb7185]" /><span className="size-2 rounded-full bg-[#fbbf24]" /><span className="size-2 rounded-full bg-[#c8ff61]" /><span className="ml-3 font-mono text-[11px] text-[#a3a8a4]">openport / ports</span></div><span className="font-mono text-[10px] uppercase tracking-[.12em] text-[#a3a8a4]">Interactive preview</span></div>
      <div className="flex min-w-0 flex-wrap border-b border-white/10"><span className="flex min-h-11 items-center gap-2 border-r border-[#c8ff61]/30 bg-[#c8ff61]/10 px-4 font-mono text-xs text-[#c8ff61]"><span className="text-[#a3a8a4]">01</span> Ports</span><span className="flex min-h-11 items-center px-4 font-mono text-[11px] text-[#a3a8a4]">LOCAL MACHINE · SAMPLE DATA</span></div>
      <div className="grid min-w-0 lg:grid-cols-[minmax(0,1fr)_210px] xl:grid-cols-[minmax(0,1fr)_240px]">
        <div className="min-w-0 p-4 sm:p-5">
          <div className="mb-5 flex min-w-0 flex-wrap items-center justify-between gap-3"><div className="flex min-w-0 flex-wrap items-center gap-2"><span className="size-2 rounded-full bg-[#c8ff61] shadow-[0_0_12px_#c8ff61]" /><span className="font-mono text-xs text-[#f5f7f4]">{ports.length} ports listening</span></div><span className="font-mono text-[10px] text-[#a3a8a4]">SORT: PORT ↑</span></div>
          <label className="mb-5 flex min-h-11 min-w-0 flex-wrap items-center gap-2 rounded-lg border border-white/15 bg-[#070908] px-3 text-[#a3a8a4] focus-within:border-[#c8ff61]/70"><Search className="size-4 shrink-0" /><span className="sr-only">Filter sample ports</span><input className="min-w-0 flex-1 bg-transparent font-mono text-xs text-[#f5f7f4] outline-none placeholder:text-[#a3a8a4]" value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Filter port or process..." type="search" /></label>
          <div className="grid grid-cols-[minmax(55px,.65fr)_minmax(0,1fr)_minmax(0,1fr)] gap-2 border-b border-white/10 px-2 pb-2 font-mono text-[10px] uppercase tracking-[.12em] text-[#a3a8a4] sm:grid-cols-[minmax(65px,.65fr)_minmax(0,1fr)_minmax(0,1fr)_50px]"><span>Port</span><span>Process</span><span>Alias</span><span className="hidden sm:block">State</span></div>
          <div className="mt-1 min-h-[208px]">{filtered.length ? filtered.map((item) => <button key={item.port} type="button" aria-pressed={selected === item.port} onClick={() => selectPort(item.port)} className={`grid min-h-[52px] w-full min-w-0 grid-cols-[minmax(55px,.65fr)_minmax(0,1fr)_minmax(0,1fr)] items-center gap-2 rounded-md border-l-2 px-2 text-left font-mono text-[11px] transition focus-visible:outline-2 focus-visible:outline-[#c8ff61] active:bg-[#c8ff61]/15 sm:grid-cols-[minmax(65px,.65fr)_minmax(0,1fr)_minmax(0,1fr)_50px] ${selected === item.port ? "border-[#c8ff61] bg-[#c8ff61]/[.09] text-[#f5f7f4]" : "border-transparent text-[#a3a8a4] hover:bg-white/[.06] hover:text-[#f5f7f4]"}`}><span className="text-[#c8ff61]">:{item.port}</span><span className="min-w-0 break-all">{item.process}</span><span className="min-w-0 break-all">{item.name}</span><span className="hidden text-[9px] text-[#a3a8a4] sm:block">LISTEN</span></button>) : <div className="flex min-h-36 items-center justify-center px-3 text-center font-mono text-xs text-[#a3a8a4]">No sample ports match “{query}”.</div>}</div>
        </div>
        <div className="min-w-0 border-t border-white/10 bg-[#0d100e] p-5 lg:border-l lg:border-t-0"><div className="mb-5 flex flex-wrap items-center justify-between gap-2"><span className="font-mono text-[10px] uppercase tracking-[.12em] text-[#a3a8a4]">Selected service</span><span className="text-[#c8ff61]">↗</span></div><AnimatePresence mode="wait"><motion.div key={current.port} initial={{ opacity: 0, y: 5 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -5 }} transition={{ duration: .16 }}><h3 className="mb-1 min-w-0 break-all font-display text-xl font-bold tracking-[-.04em] text-[#f5f7f4]">{current.name}</h3><p className="mb-5 font-mono text-xs text-[#a3a8a4]">:{current.port} / tcp</p><dl className="space-y-3 border-y border-white/10 py-4 font-mono text-xs">{[["PROCESS", current.process], ["PID", current.pid], ["BIND", current.address], ["HEALTH", checking ? "Checking…" : current.health]].map(([label, value]) => <div key={label} className="flex min-w-0 flex-wrap justify-between gap-2"><dt className="text-[#a3a8a4]">{label}</dt><dd className="min-w-0 break-all text-[#f5f7f4]">{value}</dd></div>)}</dl></motion.div></AnimatePresence><div className="mt-4 flex flex-wrap gap-2"><button type="button" disabled={checking} onClick={() => { setNotice(""); setChecking(true) }} className="preview-action disabled:cursor-wait disabled:opacity-50"><Zap className="size-3.5" />{checking ? "Checking…" : "Check health"}</button><button type="button" onClick={toggleProtection} className="preview-action"><LockKeyhole className="size-3.5" />{current.protected ? "Unprotect" : "Protect"}</button></div><p role="status" aria-live="polite" className="mt-3 min-h-8 break-words font-mono text-[10px] leading-relaxed text-[#c8ff61]">{notice || (current.protected ? "Protected from destructive actions." : "Select a port to inspect it.")}</p></div>
      </div>
      <div className="flex min-w-0 flex-wrap items-center justify-between gap-2 border-t border-white/10 bg-[#121613] px-4 py-3 font-mono text-[10px] text-[#a3a8a4] sm:px-5"><span>THIS IS A PREVIEW — NOT YOUR MACHINE</span><span>↑↓ navigate in the real CLI</span></div>
    </div>
  )
}

function App() {
  return <div id="top" className="site min-h-screen overflow-x-hidden bg-[#070908] text-[#f5f7f4]"><Header /><main>
    <section className="hero-grid relative overflow-hidden border-b border-white/10"><div className="relative mx-auto grid max-w-[1440px] min-w-0 lg:grid-cols-[minmax(0,.9fr)_minmax(0,1.1fr)]"><div className="relative flex min-w-0 flex-col justify-center px-5 pb-10 pt-16 sm:px-8 sm:pt-20 lg:min-h-[740px] lg:border-r lg:border-white/10 lg:px-12 lg:pb-20 lg:pt-24"><div className="absolute left-0 top-0 hidden h-1 w-24 bg-[#c8ff61] lg:block" /><div className="mb-10 flex min-w-0 flex-wrap items-center gap-3 font-mono text-[11px] uppercase tracking-[.15em] text-[#c8ff61]"><span className="inline-flex size-6 items-center justify-center rounded-full border border-[#c8ff61]/40">01</span> The local development cockpit <span className="h-px w-12 bg-[#c8ff61]/40" /></div><h1 className="max-w-[720px] min-w-0 break-words font-display text-[clamp(3.3rem,7vw,7.3rem)] font-bold leading-[.94] tracking-[-.075em] text-[#f5f7f4]">Know what’s<br />on <span className="relative inline-block text-[#c8ff61]">your port<span className="absolute -bottom-2 left-0 h-[3px] w-full bg-[#c8ff61]" /></span><span className="text-[#f5f7f4]">.</span></h1><p className="mt-10 max-w-[460px] text-base leading-[1.7] text-[#c5cac5] sm:text-lg">Port already in use? See the process behind it, clear the conflict, and get back to building. No command archaeology required.</p><div className="mt-8 flex min-w-0 flex-wrap gap-3"><a href="#install" className="button-lime min-h-12 px-6">Install OpenPort <ArrowRight className="size-4" /></a><a href="#how-it-works" className="button-outline min-h-12 px-6">See how it works <ArrowDownRight className="size-4" /></a></div><div className="mt-12 flex min-w-0 flex-wrap gap-x-5 gap-y-3 border-t border-white/10 pt-5 font-mono text-[10px] uppercase tracking-[.12em] text-[#a3a8a4]"><span><span className="mr-2 text-[#c8ff61]">●</span> Open source</span><span><span className="mr-2 text-[#c8ff61]">●</span> Offline first</span><span><span className="mr-2 text-[#c8ff61]">●</span> macOS · Linux · Windows</span></div></div><div className="relative flex min-w-0 items-center overflow-hidden border-t border-white/10 px-5 pb-16 pt-8 sm:px-8 lg:border-t-0 lg:px-10 lg:py-16"><div className="pointer-events-none absolute -right-28 -top-20 size-[500px] rounded-full bg-[#c8ff61]/[.065] blur-[110px]" /><div className="relative w-full min-w-0"><div className="mb-4 flex min-w-0 flex-wrap items-center justify-between gap-3 font-mono text-[10px] uppercase tracking-[.14em] text-[#a3a8a4]"><span>Fig. 01 / Find the thing in your way</span><span className="text-[#c8ff61]">● IN FOCUS</span></div><ConsolePreview /><p className="mt-4 flex min-w-0 flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-[.1em] text-[#a3a8a4]"><span className="h-px w-8 bg-[#c8ff61]" /> A preview of the real terminal experience</p></div></div></div></section>
    <section className="border-b border-white/10 bg-[#c8ff61] text-[#070908]"><div className="mx-auto flex max-w-[1440px] flex-wrap items-center justify-between gap-x-6 gap-y-2 px-5 py-5 sm:px-8 lg:px-12"><p className="font-display text-lg font-bold tracking-[-.04em] sm:text-xl">One less thing between you and shipping.</p><span className="font-mono text-xs font-medium uppercase tracking-[.12em]">Your machine. Your rules. ↗</span></div></section>
    <section id="features" className="scroll-mt-8 border-b border-white/10 px-5 py-20 sm:px-8 sm:py-28 lg:px-12"><div className="mx-auto max-w-[1344px]"><div className="mb-12 flex min-w-0 flex-wrap items-end justify-between gap-8"><div><p className="eyebrow">01 / CAPABILITIES</p><h2 className="mt-5 max-w-[750px] font-display text-[clamp(2.5rem,5vw,5.4rem)] font-bold leading-[1.02] tracking-[-.065em]">Less detective work.<br /><span className="text-[#a3a8a4]">More momentum.</span></h2></div><p className="max-w-[270px] border-l-2 border-[#c8ff61] pl-4 text-sm leading-relaxed text-[#c5cac5]">Everything you need when a port conflict interrupts your flow.</p></div><div className="grid gap-px overflow-hidden rounded-2xl border border-white/10 bg-white/10 md:grid-cols-3">{[
      { num: "01", icon: Search, title: "See the whole picture.", description: "Live ports, owning processes, PIDs, and bind addresses. The answer is already on screen.", tag: "VISIBILITY" },
      { num: "02", icon: Zap, title: "Act with confidence.", description: "Check health, find a free port, or stop a conflicting process—with a confirmation before anything destructive.", tag: "CONTROL" },
      { num: "03", icon: ShieldCheck, title: "Keep the important stuff safe.", description: "Name familiar services with aliases and protect ports you never want to stop by accident.", tag: "PEACE OF MIND" },
    ].map(({ num, icon: Icon, title, description, tag }) => <article key={num} className="group flex min-w-0 flex-col bg-[#0d100e] p-7 transition-colors duration-200 hover:bg-[#121613] sm:p-9"><div className="mb-20 flex flex-wrap items-start justify-between gap-3"><span className="flex size-12 items-center justify-center rounded-xl border border-[#c8ff61]/30 bg-[#c8ff61]/[.08] text-[#c8ff61]"><Icon className="size-5" strokeWidth={1.7} /></span><span className="font-mono text-xs text-[#a3a8a4]">/{num}</span></div><span className="mb-4 font-mono text-[10px] tracking-[.15em] text-[#c8ff61]">{tag}</span><h3 className="mb-4 break-words font-display text-2xl font-bold tracking-[-.045em]">{title}</h3><p className="max-w-sm text-sm leading-[1.75] text-[#c5cac5]">{description}</p></article>)}</div></div></section>
    <section id="how-it-works" className="scroll-mt-8 overflow-hidden border-b border-white/10 bg-[#0a0d0b] px-5 py-20 sm:px-8 sm:py-28 lg:px-12"><div className="mx-auto grid max-w-[1344px] min-w-0 gap-12 lg:grid-cols-[minmax(0,.85fr)_minmax(0,1.15fr)] lg:gap-20"><div className="min-w-0"><p className="eyebrow">02 / THE WORKFLOW</p><h2 className="mt-5 max-w-[570px] font-display text-[clamp(2.5rem,5vw,5.4rem)] font-bold leading-[1.02] tracking-[-.065em]">From error<br />to <span className="text-[#c8ff61]">answer.</span></h2><p className="mt-7 max-w-[440px] leading-[1.75] text-[#c5cac5]">The next time you hit <code className="break-all rounded bg-white/[.07] px-1.5 py-1 font-mono text-sm text-[#f5f7f4]">EADDRINUSE</code>, skip the hunt. OpenPort turns a blocked port into a clear next step.</p><a href="#install" className="mt-9 inline-flex min-h-11 flex-wrap items-center gap-3 border-b border-[#c8ff61] text-sm font-semibold text-[#c8ff61] transition hover:gap-5 hover:text-[#d7ff8a] focus-visible:outline-2 focus-visible:outline-[#c8ff61]">Get up and running <ArrowRight className="size-4" /></a></div><div className="min-w-0 border-t border-white/15">{[
      { n: "01", title: "Launch", body: "Run openport. Your local listeners appear in a live, keyboard-driven view.", icon: Terminal },
      { n: "02", title: "Understand", body: "Find the port, inspect the process behind it, and give familiar services a name.", icon: CircleHelp },
      { n: "03", title: "Move forward", body: "Check health or resolve a conflict. Protected ports and confirmations keep you in charge.", icon: Shield },
    ].map(({ n, title, body, icon: Icon }) => <div key={n} className="grid min-w-0 grid-cols-[48px_minmax(0,1fr)] gap-4 border-b border-white/15 py-7 sm:grid-cols-[64px_minmax(0,1fr)_44px] sm:gap-6"><span className="pt-1 font-mono text-sm text-[#c8ff61]">{n}/</span><div className="min-w-0"><h3 className="font-display text-2xl font-bold tracking-[-.04em]">{title}</h3><p className="mt-2 max-w-[440px] break-words text-sm leading-[1.75] text-[#c5cac5]">{body}</p></div><Icon className="hidden size-6 text-[#a3a8a4] sm:block" strokeWidth={1.5} /></div>)}</div></div></section>
    <section id="install" className="scroll-mt-8 px-5 py-20 sm:px-8 sm:py-28 lg:px-12"><div className="mx-auto max-w-[1344px] overflow-hidden rounded-2xl border border-[#c8ff61]/40 bg-[#c8ff61] text-[#070908]"><div className="grid min-w-0 lg:grid-cols-[minmax(0,1fr)_minmax(0,.85fr)]"><div className="min-w-0 p-7 sm:p-12 lg:p-16"><p className="font-mono text-xs font-semibold uppercase tracking-[.15em]">03 / GET STARTED</p><h2 className="mt-8 max-w-[570px] font-display text-[clamp(2.5rem,5vw,5.4rem)] font-bold leading-[1.02] tracking-[-.065em]">Take back<br />port 3000.</h2><p className="mt-6 max-w-md text-base leading-relaxed text-[#26331b]">One command to install. One command to see what’s running. No account, no cloud, no guesswork.</p><a href={repo} target="_blank" rel="noopener noreferrer" className="mt-8 inline-flex min-h-11 flex-wrap items-center gap-2 border-b border-[#070908] text-sm font-bold transition hover:gap-4 focus-visible:outline-2 focus-visible:outline-[#070908]">View source on GitHub <ArrowRight className="size-4" /></a></div><div className="flex min-w-0 flex-col justify-center border-t border-[#070908]/20 bg-[#b9ee59] p-7 sm:p-12 lg:border-l lg:border-t-0 lg:p-14"><div className="mb-5 flex flex-wrap items-center justify-between gap-3 font-mono text-[11px] font-semibold uppercase tracking-[.12em]"><span>INSTALL / GO 1.25+</span><span>01 — COPY</span></div><CopyInstall light /><div className="mt-7 border-t border-[#070908]/20 pt-7"><p className="mb-3 font-mono text-[11px] font-semibold uppercase tracking-[.12em]">02 — LAUNCH</p><div className="rounded-xl bg-[#070908] px-5 py-4 font-mono text-sm text-[#f5f7f4]"><span className="mr-3 text-[#c8ff61]">$</span>openport<span className="ml-1 inline-block h-4 w-2 animate-pulse bg-[#c8ff61] align-[-3px] motion-reduce:animate-none" /></div></div><p className="mt-6 flex min-w-0 flex-wrap items-center gap-2 text-xs font-medium text-[#26331b]"><CheckCheck className="size-4" /> Available on macOS, Linux & Windows</p></div></div></div></section>
  </main><footer className="border-t border-white/10 px-5 py-8 sm:px-8 lg:px-12"><div className="mx-auto flex max-w-[1344px] min-w-0 flex-wrap items-center justify-between gap-5"><a href="#top" className="flex min-h-11 flex-wrap items-center gap-3 font-display text-lg font-bold tracking-[-.05em] text-[#f5f7f4] hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]"><Mark />openport</a><p className="text-sm text-[#a3a8a4]">Built for the moments between an error and the fix.</p><a href={repo} target="_blank" rel="noopener noreferrer" className="flex min-h-11 flex-wrap items-center gap-2 text-sm text-[#c5cac5] transition hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]"><Github className="size-4" /> GitHub <ChevronRight className="size-4" /></a></div></footer></div>
}

export default App
