import { motion } from "framer-motion";
import { Activity, CheckCircle2, XCircle, Gauge } from "lucide-react";
import type { Stats } from "../types";

interface StatsGridProps {
  stats: Stats | null;
}

export function StatsGrid({ stats }: StatsGridProps) {
  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: 0.1
      }
    }
  };

  const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { 
      opacity: 1, 
      y: 0,
      transition: {
        duration: 0.5,
        ease: [0.4, 0, 0.2, 1] as const
      }
    }
  };

  return (
    <motion.div 
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"
    >
      <motion.div
        variants={itemVariants}
        className="relative group bg-[#12151a] border border-[#22262f] p-5 flex flex-col"
      >
        <div className="flex items-center gap-2 mb-4 text-[#94a3b8]">
          <Activity className="h-4 w-4" />
          <span className="text-xs font-mono tracking-widest uppercase">Global Uptime</span>
        </div>
        <p className="text-3xl font-bold font-mono text-[#e2e8f0]">
          {stats ? Math.round(stats.overallUptime) : 0}%
        </p>
      </motion.div>

      <motion.div
        variants={itemVariants}
        className="relative group bg-[#12151a] border border-[#22262f] p-5 flex flex-col"
      >
        <div className="flex items-center gap-2 mb-4 text-[#94a3b8]">
          <CheckCircle2 className="h-4 w-4 text-[#00e59b]" />
          <span className="text-xs font-mono tracking-widest uppercase">Online Services</span>
        </div>
        <p className="text-3xl font-bold font-mono text-[#00e59b]">
          {stats?.servicesUp || 0}
        </p>
      </motion.div>

      <motion.div
        variants={itemVariants}
        className="relative group bg-[#12151a] border border-[#22262f] p-5 flex flex-col"
      >
        <div className="flex items-center gap-2 mb-4 text-[#94a3b8]">
          <XCircle className="h-4 w-4 text-[#ff3366]" />
          <span className="text-xs font-mono tracking-widest uppercase">Offline Services</span>
        </div>
        <p className="text-3xl font-bold font-mono text-[#ff3366]">
          {stats?.servicesDown || 0}
        </p>
      </motion.div>

      <motion.div
        variants={itemVariants}
        className="relative group bg-[#12151a] border border-[#22262f] p-5 flex flex-col"
      >
        <div className="flex items-center gap-2 mb-4 text-[#94a3b8]">
          <Gauge className="h-4 w-4 text-[#e2e8f0]" />
          <span className="text-xs font-mono tracking-widest uppercase">Avg Latency</span>
        </div>
        <p className="text-3xl font-bold font-mono text-[#e2e8f0]">
          {stats?.avgResponseTime || 0}ms
        </p>
      </motion.div>
    </motion.div>
  );
}

