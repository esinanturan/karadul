import { useState } from "react"
import {
  Settings,
  Key,
  Shield,
  Copy,
  Trash2,
  Plus,
  RefreshCw,
  Clock,
} from "lucide-react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuthKeys, useCreateAuthKey, useDeleteAuthKey } from "@/lib/api"
import { formatDate, cn } from "@/lib/utils"
import { ErrorAlert } from "@/components/error-boundary"
import { EmptyState } from "@/components/empty-state"
import { toast } from "sonner"

function AuthKeySkeleton() {
  return (
    <TableRow>
      <TableCell><Skeleton className="h-4 w-32" /></TableCell>
      <TableCell><Skeleton className="h-4 w-20" /></TableCell>
      <TableCell><Skeleton className="h-4 w-16" /></TableCell>
      <TableCell><Skeleton className="h-4 w-20" /></TableCell>
      <TableCell><Skeleton className="h-8 w-8" /></TableCell>
    </TableRow>
  )
}

export function SettingsPage() {
  const { data: authKeys, isLoading, error, refetch } = useAuthKeys()
  const createAuthKey = useCreateAuthKey()
  const deleteAuthKey = useDeleteAuthKey()
  const [newKeyExpiresIn, setNewKeyExpiresIn] = useState<string>("")
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [copiedKey, setCopiedKey] = useState<string | null>(null)

  const handleCreateKey = async () => {
    try {
      await createAuthKey.mutateAsync(newKeyExpiresIn || undefined)
      toast.success("Authentication key created successfully")
      setShowCreateDialog(false)
      setNewKeyExpiresIn("")
    } catch (err) {
      toast.error(`Failed to create key: ${err instanceof Error ? err.message : "Unknown error"}`)
    }
  }

  const handleCopyKey = (key: string) => {
    navigator.clipboard.writeText(key)
    setCopiedKey(key)
    toast.success("Key copied to clipboard")
    setTimeout(() => setCopiedKey(null), 2000)
  }

  const handleDeleteKey = async (id: string) => {
    try {
      await deleteAuthKey.mutateAsync(id)
      toast.success("Authentication key deleted successfully")
    } catch (err) {
      toast.error(`Failed to delete key: ${err instanceof Error ? err.message : "Unknown error"}`)
    }
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Settings</h2>
          <p className="text-muted-foreground">
            Configure your Karadul mesh network
          </p>
        </div>
        <ErrorAlert
          title="Failed to load settings"
          message={error.message}
          onRetry={() => refetch()}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Settings</h2>
          <p className="text-muted-foreground">
            Configure your Karadul mesh network
          </p>
        </div>
        <Button
          variant="outline"
          size="icon"
          onClick={() => refetch()}
          disabled={isLoading}
        >
          <RefreshCw className={cn("h-4 w-4", isLoading && "animate-spin")} />
        </Button>
      </div>

      <Tabs defaultValue="auth-keys" className="w-full">
        <TabsList>
          <TabsTrigger value="auth-keys">
            <Key className="h-4 w-4 mr-2" />
            Auth Keys
          </TabsTrigger>
          <TabsTrigger value="acl">
            <Shield className="h-4 w-4 mr-2" />
            ACL Rules
          </TabsTrigger>
          <TabsTrigger value="general">
            <Settings className="h-4 w-4 mr-2" />
            General
          </TabsTrigger>
        </TabsList>

        <TabsContent value="auth-keys" className="space-y-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Authentication Keys</CardTitle>
                <CardDescription>
                  Manage authentication keys for new nodes
                </CardDescription>
              </div>
              <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
                <DialogTrigger asChild>
                  <Button>
                    <Plus className="h-4 w-4 mr-2" />
                    Create Key
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Create Auth Key</DialogTitle>
                    <DialogDescription>
                      Create a new authentication key for node enrollment
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    <div className="space-y-2">
                      <Label>Expiration</Label>
                      <Select
                        value={newKeyExpiresIn}
                        onValueChange={setNewKeyExpiresIn}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Never expires" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="">Never expires</SelectItem>
                          <SelectItem value="1h">1 hour</SelectItem>
                          <SelectItem value="24h">24 hours</SelectItem>
                          <SelectItem value="7d">7 days</SelectItem>
                          <SelectItem value="30d">30 days</SelectItem>
                          <SelectItem value="90d">90 days</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <DialogFooter>
                    <Button
                      variant="outline"
                      onClick={() => setShowCreateDialog(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      onClick={handleCreateKey}
                      disabled={createAuthKey.isPending}
                    >
                      {createAuthKey.isPending ? (
                        <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                      ) : (
                        <Plus className="h-4 w-4 mr-2" />
                      )}
                      Create
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Key</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead>Expires</TableHead>
                      <TableHead>Used By</TableHead>
                      <TableHead className="w-[100px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Array.from({ length: 3 }).map((_, i) => (
                      <AuthKeySkeleton key={i} />
                    ))}
                  </TableBody>
                </Table>
              ) : authKeys?.length === 0 ? (
                <EmptyState
                  icon={Key}
                  title="No auth keys"
                  description="Create an authentication key to allow new nodes to join your mesh network."
                  action={{
                    label: "Create Key",
                    onClick: () => setShowCreateDialog(true),
                  }}
                />
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Key</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead>Expires</TableHead>
                      <TableHead>Used By</TableHead>
                      <TableHead className="w-[100px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {authKeys?.map((authKey) => (
                      <TableRow key={authKey.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <code className="bg-muted px-2 py-1 rounded text-xs font-mono">
                              {authKey.key.slice(0, 20)}...
                            </code>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-6 w-6"
                              onClick={() => handleCopyKey(authKey.key)}
                            >
                              <Copy className="h-3 w-3" />
                            </Button>
                            {copiedKey === authKey.key && (
                              <Badge variant="outline" className="text-xs">
                                Copied!
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>{formatDate(authKey.createdAt)}</TableCell>
                        <TableCell>
                          {authKey.expiresAt ? (
                            <div className="flex items-center gap-1 text-amber-600">
                              <Clock className="h-3 w-3" />
                              {formatDate(authKey.expiresAt)}
                            </div>
                          ) : (
                            <Badge variant="outline">Never</Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          {authKey.usedBy ? (
                            <Badge>{authKey.usedBy}</Badge>
                          ) : (
                            <span className="text-muted-foreground">
                              Unused
                            </span>
                          )}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleDeleteKey(authKey.id)}
                            disabled={deleteAuthKey.isPending}
                          >
                            <Trash2 className="h-4 w-4 text-red-500" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="acl" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Access Control Rules</CardTitle>
              <CardDescription>
                Configure network access control rules
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-center py-8 text-muted-foreground">
                ACL configuration coming soon
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="general" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>General Settings</CardTitle>
              <CardDescription>
                Configure general network settings
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-2">
                <Label htmlFor="network-name">Network Name</Label>
                <Input
                  id="network-name"
                  placeholder="karadul"
                  defaultValue="karadul"
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="coord-url">Coordinator URL</Label>
                <Input
                  id="coord-url"
                  placeholder="https://coord.example.com"
                />
              </div>
              <Button>Save Changes</Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
