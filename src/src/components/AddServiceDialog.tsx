import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { NewService } from "../types";

interface AddServiceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  newService: NewService;
  onServiceChange: (service: NewService) => void;
  onCreate: () => void;
}

export function AddServiceDialog({
  open,
  onOpenChange,
  newService,
  onServiceChange,
  onCreate,
}: AddServiceDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px] bg-[#12151a] border border-[#22262f] rounded-none p-6">
        <DialogHeader>
          <DialogTitle className="text-[#e2e8f0] font-bold text-xl">Add New Service</DialogTitle>
          <DialogDescription className="text-[#94a3b8] font-mono text-xs">
            Add a new service to monitor. Enter the service name and URL.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="name" className="text-[#e2e8f0] text-xs font-mono uppercase tracking-wider">Service Name</Label>
            <Input
              id="name"
              placeholder="My Service"
              value={newService.name}
              onChange={(e) => onServiceChange({ ...newService, name: e.target.value })}
              className="bg-[#0a0c10] border-[#22262f] text-[#e2e8f0] rounded-none focus-visible:ring-1 focus-visible:ring-[#00e59b]"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="url" className="text-[#e2e8f0] text-xs font-mono uppercase tracking-wider">URL</Label>
            <Input
              id="url"
              placeholder="https://example.com"
              value={newService.url}
              onChange={(e) => onServiceChange({ ...newService, url: e.target.value })}
              className="bg-[#0a0c10] border-[#22262f] text-[#e2e8f0] font-mono text-sm rounded-none focus-visible:ring-1 focus-visible:ring-[#00e59b]"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="icon" className="text-[#e2e8f0] text-xs font-mono uppercase tracking-wider">Icon (optional)</Label>
            <Input
              id="icon"
              placeholder="📧"
              value={newService.icon}
              onChange={(e) => onServiceChange({ ...newService, icon: e.target.value })}
              className="bg-[#0a0c10] border-[#22262f] text-[#e2e8f0] rounded-none focus-visible:ring-1 focus-visible:ring-[#00e59b]"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="checkInterval" className="text-[#e2e8f0] text-xs font-mono uppercase tracking-wider">Check Interval (seconds)</Label>
            <Input
              id="checkInterval"
              type="number"
              min="10"
              max="3600"
              placeholder="60"
              value={newService.checkInterval}
              onChange={(e) => onServiceChange({ ...newService, checkInterval: parseInt(e.target.value) || 60 })}
              className="bg-[#0a0c10] border-[#22262f] text-[#e2e8f0] font-mono rounded-none focus-visible:ring-1 focus-visible:ring-[#00e59b]"
            />
            <p className="text-[10px] font-mono text-[#94a3b8]">
              How often to check this service (10-3600 seconds, default: 60)
            </p>
          </div>
          <div className="flex items-center space-x-2 mt-2">
            <input
              type="checkbox"
              id="thirdParty"
              checked={newService.isThirdParty}
              onChange={(e) => onServiceChange({ ...newService, isThirdParty: e.target.checked })}
              className="rounded-none border-[#22262f] bg-[#0a0c10] text-[#00e59b] focus:ring-[#00e59b]"
            />
            <Label htmlFor="thirdParty" className="text-xs font-mono text-[#94a3b8] uppercase tracking-wider cursor-pointer">
              Third-party service
            </Label>
          </div>
        </div>
        <DialogFooter className="gap-2 sm:gap-0 mt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)} className="rounded-none border-[#22262f] bg-transparent text-[#94a3b8] hover:bg-[#22262f] hover:text-[#e2e8f0] font-mono text-xs uppercase tracking-wider">
            Cancel
          </Button>
          <Button 
            onClick={onCreate}
            disabled={!newService.name || !newService.url}
            className="rounded-none bg-[#00e59b] hover:bg-[#00e59b]/80 text-[#0a0c10] font-mono text-xs font-bold uppercase tracking-wider"
          >
            Add Service
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

