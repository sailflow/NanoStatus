import { memo } from "react";
import { motion } from "framer-motion";
import { CheckCircle2, XCircle, Server, Pause } from "lucide-react";
import type { Monitor } from "../types";

interface ServiceCardProps {
  monitor: Monitor;
  isSelected: boolean;
  onClick: () => void;
  index: number;
}

export const ServiceCard = memo(function ServiceCard({ monitor, isSelected, onClick, index }: ServiceCardProps) {
  const isPaused = monitor.paused || false;

  return (
    <motion.div
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, scale: 0.98 }}
      transition={{ delay: index * 0.03, duration: 0.2 }}
      onClick={onClick}
      className={`group cursor-pointer border-l-2 border-y border-r transition-colors flex flex-col ${
        isSelected
          ? "border-l-[#00e59b] border-y-[#22262f] border-r-[#22262f] bg-[#12151a]"
          : "border-l-transparent border-y-transparent border-r-transparent hover:bg-[#12151a] hover:border-y-[#22262f] hover:border-r-[#22262f]"
      }`}
    >
      <div className="flex items-center justify-between p-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className={`h-2 w-2 rounded-full flex-shrink-0 ${
            isPaused ? 'bg-[#94a3b8]' : monitor.status === 'up' ? 'bg-[#00e59b]' : 'bg-[#ff3366]'
          }`} />
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-sm text-[#e2e8f0] truncate">{monitor.name}</h3>
            <p className="text-[10px] font-mono text-[#94a3b8] truncate">{monitor.url}</p>
          </div>
        </div>
        
        <div className="flex flex-col items-end flex-shrink-0 ml-3">
          {!isPaused && monitor.status === "up" ? (
            <span className="text-sm font-bold font-mono text-[#e2e8f0]">{monitor.responseTime}ms</span>
          ) : isPaused ? (
            <span className="text-xs font-mono text-[#94a3b8] uppercase tracking-wider">Paused</span>
          ) : (
            <span className="text-xs font-mono text-[#ff3366] uppercase tracking-wider">Down</span>
          )}
          <span className="text-[10px] font-mono text-[#94a3b8]">
            {Math.round(monitor.uptime)}% UP
          </span>
        </div>
      </div>
      
      {/* Mini sparkline equivalent - just a flat bar for uptime visual */}
      <div className="w-full h-[2px] bg-[#22262f]">
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${monitor.uptime}%` }}
          transition={{ duration: 0.5, ease: "easeOut" }}
          className={`h-full ${
            isPaused
              ? "bg-[#94a3b8]"
              : monitor.uptime > 99 ? "bg-[#00e59b]" :
              monitor.uptime === 0 ? "bg-[#ff3366]" :
              "bg-[#00e59b]/50"
          }`}
        />
      </div>
    </motion.div>
  );
}, (prev, next) => {
  // Only re-render if visual properties changed (ignoring new onClick closures)
  return prev.isSelected === next.isSelected &&
         prev.index === next.index &&
         prev.monitor.status === next.monitor.status &&
         prev.monitor.responseTime === next.monitor.responseTime &&
         prev.monitor.uptime === next.monitor.uptime &&
         prev.monitor.paused === next.monitor.paused &&
         prev.monitor.name === next.monitor.name;
});

