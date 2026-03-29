import { Moon, Sun, Bell, Wifi, WifiOff, Menu, Network, LayoutDashboard, Server, Users, Settings, Github } from "lucide-react"
import { Link, useLocation } from "react-router-dom"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { useWebSocket } from "@/lib/websocket"
import { cn } from "@/lib/utils"
import { useState } from "react"

interface HeaderProps {
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

export function Header({ className }: HeaderProps) {
  const { theme, setTheme } = useTheme()
  const { connected } = useWebSocket()
  const location = useLocation()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  return (
    <header
      className={cn(
        "flex h-16 items-center justify-between border-b px-6",
        className
      )}
    >
      <div className="flex items-center gap-4">
        <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon" className="lg:hidden">
              <Menu className="h-5 w-5" />
              <span className="sr-only">Open menu</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="w-64 p-0">
            <SheetHeader className="px-4 py-6 text-left">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
                  <Network className="h-5 w-5 text-primary-foreground" />
                </div>
                <SheetTitle>Karadul</SheetTitle>
              </div>
            </SheetHeader>
            <Separator />
            <nav className="flex-1 px-2 py-4 space-y-1">
              {navItems.map((item) => (
                <Link
                  key={item.href}
                  to={item.href}
                  onClick={() => setMobileMenuOpen(false)}
                >
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
                href="https://github.com/karadul/karadul"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="ghost" className="w-full justify-start gap-2">
                  <Github className="h-4 w-4" />
                  GitHub
                </Button>
              </a>
            </div>
          </SheetContent>
        </Sheet>

        <h1 className="text-xl font-semibold">Karadul Mesh VPN</h1>
        <Badge variant={connected ? "default" : "destructive"} className="gap-1">
          {connected ? (
            <>
              <Wifi className="h-3 w-3" />
              Connected
            </>
          ) : (
            <>
              <WifiOff className="h-3 w-3" />
              Disconnected
            </>
          )}
        </Badge>
      </div>

      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon">
          <Bell className="h-5 w-5" />
          <span className="sr-only">Notifications</span>
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              {theme === "light" ? (
                <Sun className="h-5 w-5" />
              ) : theme === "dark" ? (
                <Moon className="h-5 w-5" />
              ) : (
                <Sun className="h-5 w-5" />
              )}
              <span className="sr-only">Toggle theme</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setTheme("light")}>
              Light
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTheme("dark")}>
              Dark
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setTheme("system")}>
              System
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
