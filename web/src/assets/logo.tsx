import { type SVGProps } from "react"
import { cn } from "@/lib/utils"

export function Logo({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      id="yuxin-logo"
      viewBox="0 0 120 120"
      xmlns="http://www.w3.org/2000/svg"
      height="24"
      width="24"
      className={cn("size-6", className)}
      {...props}
    >
      <title>豫鑫 API</title>
      <defs>
        <linearGradient id="logo-bg" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#0A1628" />
          <stop offset="100%" stopColor="#0F2847" />
        </linearGradient>
        <linearGradient id="logo-glow" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#00D4FF" />
          <stop offset="50%" stopColor="#0099FF" />
          <stop offset="100%" stopColor="#6366F1" />
        </linearGradient>
      </defs>

      {/* 背景 */}
      <rect width="120" height="120" rx="24" fill="url(#logo-bg)" />

      {/* 六边形外框 */}
      <polygon
        points="60,18 95,38 95,78 60,98 25,78 25,38"
        fill="none"
        stroke="url(#logo-glow)"
        strokeWidth="4"
      />

      {/* 中心节点 */}
      <circle cx="60" cy="58" r="6" fill="#00D4FF" />

      {/* 外围节点 */}
      <circle cx="60" cy="36" r="4" fill="#0099FF" />
      <circle cx="79" cy="47" r="4" fill="#0099FF" />
      <circle cx="79" cy="69" r="4" fill="#6366F1" />
      <circle cx="60" cy="80" r="4" fill="#6366F1" />
      <circle cx="41" cy="69" r="4" fill="#0099FF" />
      <circle cx="41" cy="47" r="4" fill="#0099FF" />

      {/* 连线 */}
      <line x1="60" y1="58" x2="60" y2="36" stroke="#00D4FF" strokeWidth="2" opacity="0.7" />
      <line x1="60" y1="58" x2="79" y2="47" stroke="#00D4FF" strokeWidth="2" opacity="0.7" />
      <line x1="60" y1="58" x2="79" y2="69" stroke="#00D4FF" strokeWidth="2" opacity="0.6" />
      <line x1="60" y1="58" x2="60" y2="80" stroke="#00D4FF" strokeWidth="2" opacity="0.6" />
      <line x1="60" y1="58" x2="41" y2="69" stroke="#00D4FF" strokeWidth="2" opacity="0.7" />
      <line x1="60" y1="58" x2="41" y2="47" stroke="#00D4FF" strokeWidth="2" opacity="0.7" />

      {/* 外围连线 */}
      <line x1="60" y1="36" x2="79" y2="47" stroke="#0099FF" strokeWidth="1.2" opacity="0.4" />
      <line x1="79" y1="47" x2="79" y2="69" stroke="#0099FF" strokeWidth="1.2" opacity="0.4" />
      <line x1="79" y1="69" x2="60" y2="80" stroke="#6366F1" strokeWidth="1.2" opacity="0.4" />
      <line x1="60" y1="80" x2="41" y2="69" stroke="#6366F1" strokeWidth="1.2" opacity="0.4" />
      <line x1="41" y1="69" x2="41" y2="47" stroke="#0099FF" strokeWidth="1.2" opacity="0.4" />
      <line x1="41" y1="47" x2="60" y2="36" stroke="#0099FF" strokeWidth="1.2" opacity="0.4" />
    </svg>
  )
}
