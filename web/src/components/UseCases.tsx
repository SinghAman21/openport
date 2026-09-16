import { ArrowUpRight, FileCode2, LockKeyhole, Radar, Route } from "lucide-react"

import { guides } from "@/lib/usecases"

const icons = [Radar, FileCode2, Route, LockKeyhole]

export default function UseCases() {
  return (
    <section id="use-cases" className="scroll-mt-8 border-b border-white/10 px-5 py-20 sm:px-8 sm:py-28 lg:px-12" aria-labelledby="use-cases-title">
      <div className="mx-auto max-w-[1344px] min-w-0">
        <div className="mb-12 flex min-w-0 flex-wrap items-end justify-between gap-6">
          <div className="min-w-0"><p className="eyebrow">04 / FIELD NOTES</p><h2 id="use-cases-title" className="mt-5 max-w-[760px] font-display text-[clamp(2.5rem,5vw,5.4rem)] font-bold leading-[1.02] tracking-[-.065em]">Make more of<br /><span className="text-[#a3a8a4]">your machine.</span></h2></div>
          <p className="max-w-[300px] border-l-2 border-[#c8ff61] pl-4 text-sm leading-relaxed text-[#c5cac5]">Four small guides. Four real problems. Follow the illustrated workflow, then try the command on your own machine.</p>
        </div>
        <div className="grid min-w-0 gap-4 lg:grid-cols-2">{guides.map((guide, index) => { const Icon = icons[index]; return <article key={guide.slug} className="guide-card group flex min-w-0 flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#0d100e] transition-colors duration-200 hover:border-[#c8ff61]/40 hover:bg-[#121613]"><div className="min-w-0 flex-1 p-6 sm:p-9"><div className="mb-10 flex min-w-0 flex-wrap items-start justify-between gap-4"><span className="flex size-12 items-center justify-center rounded-xl border border-[#c8ff61]/30 bg-[#c8ff61]/[.08] text-[#c8ff61]"><Icon className="size-5" strokeWidth={1.7} /></span><span className="font-mono text-xs text-[#a3a8a4]">/ {guide.number}</span></div><div className="mb-4 flex min-w-0 flex-wrap gap-x-3 gap-y-1 font-mono text-[10px] uppercase tracking-[.13em]"><span className="text-[#c8ff61]">{guide.category}</span><span className="text-[#a3a8a4]">· {guide.time}</span></div><h3 className="max-w-md break-words font-display text-2xl font-bold leading-[1.15] tracking-[-.045em] sm:text-3xl">{guide.title}</h3><p className="mt-4 max-w-md text-sm leading-[1.75] text-[#c5cac5]">{guide.excerpt}</p></div><a href={`/usecase/${guide.slug}`} className="flex min-h-14 min-w-0 flex-wrap items-center justify-between gap-3 border-t border-white/10 px-6 py-4 font-mono text-xs font-medium text-[#c8ff61] transition-colors hover:bg-[#c8ff61]/[.08] focus-visible:outline-2 focus-visible:outline-inset focus-visible:outline-[#c8ff61] active:bg-[#c8ff61]/15 sm:px-9" aria-label={`Read guide: ${guide.title}`}><span>Read the story <span className="text-[#a3a8a4]">/ {guide.time}</span></span><ArrowUpRight className="size-4 transition-transform group-hover:translate-x-1 group-hover:-translate-y-1" /></a></article> })}</div>
      </div>
    </section>
  )
}
