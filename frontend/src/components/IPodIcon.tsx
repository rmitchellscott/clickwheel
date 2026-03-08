import { cn } from '@/lib/utils'

const icons = import.meta.glob('@/assets/ipod_icons/*.png', { eager: true, import: 'default' }) as Record<string, string>

function resolveIcon(filename: string): string | undefined {
  return icons[`/src/assets/ipod_icons/${filename}`]
}

const genericIcon = resolveIcon('iPodGeneric.png')

interface IPodIconProps {
  className?: string
  size?: number
  icon?: string
}

export function IPodIcon({ className, size = 40, icon }: IPodIconProps) {
  const src = (icon && resolveIcon(icon)) || genericIcon

  return (
    <div className={cn(className)} style={{ width: size, height: size }}>
      <img
        src={src}
        alt="iPod"
        width={size}
        height={size}
        className="object-contain"
        draggable={false}
      />
    </div>
  )
}
