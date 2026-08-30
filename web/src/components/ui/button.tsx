import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"
import * as React from "react"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-semibold transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-lime-300 disabled:pointer-events-none disabled:opacity-50 active:scale-[0.98]",
  {
    variants: {
      variant: {
        default: "bg-lime-300 text-neutral-950 shadow-[0_0_30px_rgba(190,242,100,0.15)] hover:bg-lime-200 hover:shadow-[0_0_40px_rgba(190,242,100,0.28)]",
        outline: "border border-white/12 bg-white/[0.04] text-white hover:border-white/25 hover:bg-white/[0.08]",
        ghost: "text-neutral-300 hover:bg-white/[0.06] hover:text-white",
      },
      size: {
        default: "h-11 px-6",
        sm: "h-9 px-4 text-xs",
        lg: "h-13 px-7 text-base",
        icon: "size-10 p-0",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  },
)

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(({ className, variant, size, asChild = false, ...props }, ref) => {
  const Comp = asChild ? Slot : "button"
  return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />
})
Button.displayName = "Button"

export { Button }
