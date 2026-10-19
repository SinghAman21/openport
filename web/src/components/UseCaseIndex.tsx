import { ArrowLeft, ArrowRight, ArrowUpRight, Github } from "lucide-react";
import { useEffect } from "react";

import UseCaseRows from "@/components/UseCaseRows";

export default function UseCaseIndex() {
  useEffect(() => {
    document.title = "OpenPort Field Notes — Seven ways to get unstuck";
  }, []);
  return (
    <div className="site min-h-screen overflow-x-hidden bg-[#070908] text-[#f5f7f4]">
      <header className="border-b border-white/10">
        <div className="mx-auto flex min-h-[76px] max-w-[1344px] flex-wrap items-center justify-between gap-3 px-5 sm:px-8 lg:px-12">
          <a
            href="/"
            className="inline-flex min-h-11 flex-wrap items-center gap-3 font-display text-lg font-bold tracking-[-.055em] text-[#f5f7f4] hover:text-[#c8ff61] focus-visible:outline-2 focus-visible:outline-[#c8ff61]"
          >
            <span className="brand-mark" aria-hidden="true">
              <span className="brand-mark-dot" />
              <span className="brand-mark-line" />
            </span>{" "}
            openport{" "}
            <span className="font-mono text-[10px] font-normal tracking-normal text-[#a3a8a4]">
              / FIELD NOTES
            </span>
          </a>
          <a href="/" className="nav-link gap-2">
            <ArrowLeft className="size-4" /> Back to OpenPort
          </a>
        </div>
      </header>
      <main>
        <section className="hero-grid border-b border-white/10 px-5 pb-16 pt-14 sm:px-8 sm:pb-24 sm:pt-20 lg:px-12">
          <div className="mx-auto max-w-[1344px] min-w-0">
            <p className="eyebrow">A FIELD GUIDE / 07 REAL MOMENTS</p>
            <div className="mt-7 grid min-w-0 gap-9 lg:grid-cols-[minmax(0,1fr)_minmax(0,.42fr)] lg:items-end">
              <h1 className="max-w-[920px] min-w-0 break-words font-display text-[clamp(3.2rem,7vw,7.4rem)] font-bold leading-[.96] tracking-[-.075em]">
                Your machine is
                <br />
                <span className="text-[#c8ff61]">trying to tell you.</span>
              </h1>
              <div className="max-w-[390px] min-w-0 border-l-2 border-[#c8ff61] pl-5">
                <p className="font-display text-xl font-semibold leading-snug sm:text-2xl">
                  Different problem. Same question: what’s happening on my
                  machine?
                </p>
                <p className="mt-4 text-sm leading-[1.8] text-[#c5cac5]">
                  Seven short stories about the errors, restarts and messy local
                  stacks developers actually face. Choose the moment that feels
                  familiar.
                </p>
              </div>
            </div>
          </div>
        </section>
        <section
          className="px-5 py-16 sm:px-8 sm:py-24 lg:px-12"
          aria-labelledby="index-title"
        >
          <div className="mx-auto max-w-[1344px] min-w-0">
            <div className="mb-9 flex min-w-0 flex-wrap items-end justify-between gap-6">
              <div>
                <p className="eyebrow">PICK YOUR PROBLEM</p>
                <h2
                  id="index-title"
                  className="mt-3 font-display text-3xl font-bold tracking-[-.05em] sm:text-4xl"
                >
                  Where are you stuck?
                </h2>
              </div>
              <p className="max-w-[240px] font-mono text-xs leading-relaxed text-[#a3a8a4]">
                PROBLEM → OLD APPROACH → THE OPENPORT WAY
              </p>
            </div>
            <UseCaseRows />
          </div>
        </section>
        <section className="border-y border-white/10 bg-[#c8ff61] px-5 py-8 text-[#070908] sm:px-8 lg:px-12">
          <div className="mx-auto flex max-w-[1344px] min-w-0 flex-wrap items-center justify-between gap-4">
            <p className="max-w-2xl font-display text-xl font-bold tracking-[-.035em] sm:text-2xl">
              One command to see what’s running.
            </p>
            <a
              href="/#install"
              className="inline-flex min-h-11 flex-wrap items-center gap-2 rounded-lg bg-[#070908] px-5 font-semibold text-[#f5f7f4] transition hover:bg-[#202820] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#070908]"
            >
              Get OpenPort <ArrowRight className="size-4" />
            </a>
          </div>
        </section>
      </main>
      <footer className="border-t border-white/10 px-5 py-7 sm:px-8 lg:px-12">
        <div className="mx-auto flex max-w-[1344px] min-w-0 flex-wrap items-center justify-between gap-3 text-xs text-[#a3a8a4]">
          <a
            href="/"
            className="flex min-h-11 items-center gap-2 hover:text-[#c8ff61]"
          >
            <ArrowLeft className="size-4" /> OpenPort home
          </a>
          <a
            href="https://github.com/SinghAman21/openport"
            target="_blank"
            rel="noopener noreferrer"
            className="flex min-h-11 items-center gap-2 hover:text-[#c8ff61]"
          >
            <Github className="size-4" /> Source & docs{" "}
            <ArrowUpRight className="size-3" />
          </a>
        </div>
      </footer>
    </div>
  );
}
