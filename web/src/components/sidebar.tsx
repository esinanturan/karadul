import { Link, useLocation } from "react-router-dom"
import {
  Network,
  LayoutDashboard,
  Server,
  Users,
  Settings,
  Github,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

interface SidebarProps {
  className?: string
}

const navItems = [
  {
    title: "Dashboard",
    href: "/",
    icon: LayoutDashboard,
  },
  {
    title: "Topology",
    href: "/topology",
    icon: Network,
  },
  {
    title: "Nodes",
    href: "/nodes",
    icon: Server,
  },
  {
    title: "Peers",
    href: "/peers",
    icon: Users,
  },
  {
    title: "Settings",
    href: "/settings",
    icon: Settings,
  },
]

export function Sidebar({ className }: SidebarProps) {
  const location = useLocation()

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <div className="flex items-center gap-2 px-4 py-6">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
          <Network className="h-5 w-5 text-primary-foreground" />
        </div>
        <span className="text-lg font-semibold">Karadul</span>
      </div>

      <Separator />

      <nav className="flex-1 px-2 py-4 space-y-1">
        {navItems.map((item) => (
          <Link key={item.href} to={item.href}>
            <Button
              variant={location.pathname === item.href ? "secondary" : "ghost"}
              className={cn(
                "w-full justify-start gap-2",
                location.pathname === item.href && "bg-muted"
              )}
            >
              <item.icon className="h-4 w-4" />
              {item.title}
            </Button>
          </Link>
        ))}
      </nav>

      <Separator />

      <div className="p-4">
        <a
          href="https://github.com/ersinkoc/karadul"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Button variant="ghost" className="w-full justify-start gap-2">
            <Github className="h-4 w-4" />
            GitHub
          </Button>
        </a>
      </div>
    </div>
  )
}
