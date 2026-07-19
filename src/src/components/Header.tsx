import { motion } from "framer-motion";
import { Search, Plus, Activity, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface HeaderProps {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onAddService: () => void;
  onExportMonitors: () => void;
  lastUpdate: Date;
  isHealthy?: boolean;
}

export function Header({ searchQuery, onSearchChange, onAddService, onExportMonitors, lastUpdate, isHealthy = true }: HeaderProps) {
  return (
    <motion.header 
      initial={{ y: -50, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.6, ease: "easeOut" }}
      className="sticky top-0 z-50 bg-transparent"
    >
      <div className="w-full px-4 md:px-8 py-4 flex items-center justify-between border-b border-[#22262f]/50">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-3">
            <motion.div
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              className="p-2 border border-[#22262f] bg-[#12151a]"
            >
              <Activity className="h-5 w-5 text-[#e2e8f0]" />
            </motion.div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-[#e2e8f0]">
                NanoStatus
              </h1>
            </div>
          </div>
          
          <div className="hidden md:flex items-center gap-2">
            <div className={`h-2 w-2 rounded-full ${isHealthy ? 'bg-[#00e59b] shadow-[0_0_8px_#00e59b]' : 'bg-[#ff3366] shadow-[0_0_8px_#ff3366]'}`} />
            <span className="text-sm font-semibold tracking-widest uppercase text-[#e2e8f0]">
              {isHealthy ? 'All Systems Healthy' : 'System Degraded'}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="relative group">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-[#94a3b8] transition-colors" />
            <Input
              placeholder="Search..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-9 pr-3 w-64 h-9 bg-[#12151a]/80 border-[#22262f] focus:border-[#00e59b]/50 focus:ring-1 focus:ring-[#00e59b]/20 text-[#e2e8f0] placeholder:text-[#94a3b8] rounded-none font-mono text-sm"
            />
          </div>
          <Button 
            variant="outline"
            className="rounded-none bg-[#12151a] border-[#22262f] hover:bg-[#22262f] text-[#e2e8f0] h-9 px-3 font-mono text-xs uppercase tracking-wider"
            onClick={onExportMonitors}
          >
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
          <Button 
            className="rounded-none bg-[#00e59b] hover:bg-[#00e59b]/80 text-[#0a0c10] h-9 px-4 font-mono text-xs font-bold uppercase tracking-wider"
            onClick={onAddService}
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Service
          </Button>
        </div>
      </div>
    </motion.header>
  );
}

