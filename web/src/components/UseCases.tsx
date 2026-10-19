import { ArrowRight } from "lucide-react"

import UseCaseRows from "@/components/UseCaseRows"

export default function UseCases() {
  return <section id="use-cases" className="scroll-mt-8 border-b border-white/10 px-5 py-20 sm:px-8 sm:py-28 lg:px-12" aria-labelledby="use-cases-title"><div className="mx-auto max-w-[1344px] min-w-0"><div className="mb-12 flex min-w-0 flex-wrap items-end justify-between gap-8"><div className="min-w-0"><p className="eyebrow">04 / FIELD NOTES</p><h2 id="use-cases-title" className="mt-5 max-w-[760px] font-display text-[clamp(2.5rem,5vw,5.4rem)] font-bold leading-[1.02] tracking-[-.065em]">A fix for the<br /><span className="text-[#a3a8a4]">moment you're in.</span></h2></div><p className="max-w-[280px] border-l-2 border-[#c8ff61] pl-4 text-sm leading-relaxed text-[#c5cac5]">Start with three. The full field guide has seven real developer moments.</p></div><UseCaseRows compact limit={3} /><a href="/usecase" className="mt-8 inline-flex min-h-11 flex-wrap items-center gap-3 border-b border-[#c8ff61] text-sm font-semibold text-[#c8ff61] transition hover:gap-5 focus-visible:outline-2 focus-visible:outline-[#c8ff61]">Read all 7 stories <ArrowRight className="size-4" /></a></div></section>
}
