import { ArrowUpRight } from "lucide-react";

import { guides } from "@/lib/usecases";

export default function UseCaseRows({
  compact = false,
  limit,
}: {
  compact?: boolean;
  limit?: number;
}) {
  const visibleGuides = limit ? guides.slice(0, limit) : guides;
  return (
    <div className="border-t border-white/15">
      {visibleGuides.map((guide) => (
        <article key={guide.slug} className="group border-b border-white/15">
          <a
            href={`/usecase/${guide.slug}`}
            aria-label={`Read ${guide.title}`}
            className={`grid min-h-28 min-w-0 items-center gap-x-3 gap-y-3 py-6 text-[#f5f7f4] transition-colors hover:bg-[#c8ff61]/[.045] focus-visible:outline-2 focus-visible:outline-inset focus-visible:outline-[#c8ff61] active:bg-[#c8ff61]/[.09] sm:grid-cols-[52px_minmax(0,1fr)_44px] ${compact ? "lg:grid-cols-[60px_minmax(0,.97fr)_200px_44px]" : "lg:grid-cols-[60px_minmax(0,1.2fr)_minmax(0,.85fr)_44px]"}`}
          >
            <span className="font-mono text-xs text-[#c8ff61] sm:pl-4">
              {guide.number} /
            </span>
            <div className="min-w-0">
              <p className="mb-2 font-mono text-[10px] uppercase tracking-[.14em] text-[#a3a8a4]">
                {guide.category} · {guide.time}
              </p>
              <h3 className="min-w-0 break-words font-display text-xl font-bold leading-tight tracking-[-.045em] transition-colors group-hover:text-[#c8ff61] sm:text-2xl">
                {guide.title}
              </h3>
            </div>
            <p
              className={`min-w-0 break-words text-sm leading-[1.7] text-[#a3a8a4] ${compact ? "hidden lg:block" : "sm:col-start-2 sm:col-end-3 lg:col-auto"}`}
            >
              {compact ? guide.excerpt : guide.problem}
            </p>
            <span className="ml-auto hidden size-11 shrink-0 items-center justify-center rounded-full border border-white/15 text-[#c8ff61] transition-all group-hover:translate-x-1 group-hover:-translate-y-1 group-hover:border-[#c8ff61]/50 sm:flex sm:col-start-3 sm:row-start-1 lg:col-auto lg:row-auto">
              <ArrowUpRight className="size-6" />
            </span>
          </a>
        </article>
      ))}
    </div>
  );
}
