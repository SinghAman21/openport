type BrandMarkProps = {
  className?: string
  size?: number
}

export default function BrandMark({ className = "", size = 50 }: BrandMarkProps) {
  return <img className={`brand-mark-image ${className}`.trim()} src="/openport-mark.svg" width={size} height={size} style={{ width: size, height: size }} alt="" aria-hidden="true" />
}
