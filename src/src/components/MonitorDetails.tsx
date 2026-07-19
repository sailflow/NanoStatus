import { motion, AnimatePresence } from "framer-motion";
import { useState, useEffect } from "react";
import { Pause, Play, Edit, Trash2, Zap, TrendingUp, Activity, AlertCircle, BarChart3, Globe, Clock, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid } from "recharts";
import type { Monitor, ResponseTimeData } from "../types";

interface MonitorDetailsProps {
  monitor: Monitor;
  responseTimeData: ResponseTimeData[];
  onDelete: (id: string | number) => void;
  onEdit: (monitor: Monitor) => void;
  onTogglePause: (id: string | number, paused: boolean) => void;
  onFetchResponseTime?: (monitorId: string, timeRange: string) => void;
}

export function MonitorDetails({ monitor, responseTimeData, onDelete, onEdit, onTogglePause, onFetchResponseTime }: MonitorDetailsProps) {
  const avgResponseTime = responseTimeData.length > 0
    ? Math.round(responseTimeData.reduce((sum, data) => sum + data.responseTime, 0) / responseTimeData.length)
    : 0;

  const isPaused = monitor.paused || false;
  
  // Time range state (default to 24h, but show as "12h" initially for better UX)
  const [timeRange, setTimeRange] = useState<string>("12h");
  
  // Format timestamp in user's local timezone
  const formatTime = (data: ResponseTimeData, range: string): string => {
    if (data.timestamp) {
      try {
        const date = new Date(data.timestamp);
        if (isNaN(date.getTime())) {
          // Invalid date, fallback to provided time string
          return data.time;
        }
        switch (range) {
          case "1h":
          case "12h":
          case "24h":
            return date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', hour12: true });
          case "1w":
            return date.toLocaleString('en-US', { weekday: 'short', hour: 'numeric', minute: '2-digit', hour12: true });
          case "1y":
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
          default:
            return date.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', hour12: true });
        }
      } catch (e) {
        // Error parsing date, fallback to provided time string
        return data.time;
      }
    }
    // Fallback to provided time string
    return data.time;
  };
  
  // Format response time data with local timezone (recalculates when timeRange or responseTimeData changes)
  const formattedResponseTimeData = responseTimeData.map(data => ({
    ...data,
    time: formatTime(data, timeRange)
  }));
  
  // Calculate seconds since last update
  const [secondsSinceUpdate, setSecondsSinceUpdate] = useState<number>(0);
  
  // Format time in the best unit
  const formatTimeAgo = (seconds: number): string => {
    if (seconds < 60) {
      return `${seconds}s ago`;
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60);
      return `${minutes}${minutes === 1 ? ' minute' : ' minutes'} ago`;
    } else if (seconds < 86400) {
      const hours = Math.floor(seconds / 3600);
      return `${hours}${hours === 1 ? ' hour' : ' hours'} ago`;
    } else if (seconds < 604800) {
      const days = Math.floor(seconds / 86400);
      return `${days}${days === 1 ? ' day' : ' days'} ago`;
    } else if (seconds < 2592000) {
      const weeks = Math.floor(seconds / 604800);
      return `${weeks}${weeks === 1 ? ' week' : ' weeks'} ago`;
    } else if (seconds < 31536000) {
      const months = Math.floor(seconds / 2592000);
      return `${months}${months === 1 ? ' month' : ' months'} ago`;
    } else {
      const years = Math.floor(seconds / 31536000);
      return `${years}${years === 1 ? ' year' : ' years'} ago`;
    }
  };
  
  useEffect(() => {
    const calculateSeconds = () => {
      if (monitor.updatedAt) {
        const updatedAt = new Date(monitor.updatedAt);
        const now = new Date();
        const diffSeconds = Math.floor((now.getTime() - updatedAt.getTime()) / 1000);
        setSecondsSinceUpdate(Math.max(0, diffSeconds));
      } else {
        // Fallback: parse lastCheck string
        const lastCheckStr = monitor.lastCheck || "";
        const lastCheck = lastCheckStr.toLowerCase();
        if (lastCheck === "just now" || lastCheck === "never" || lastCheck === "") {
          setSecondsSinceUpdate(0);
        } else if (lastCheck.includes("s ago")) {
          const match = lastCheck.match(/(\d+)s ago/);
          setSecondsSinceUpdate(match && match[1] ? parseInt(match[1], 10) : 0);
        } else if (lastCheck.includes("m ago")) {
          const match = lastCheck.match(/(\d+)m ago/);
          setSecondsSinceUpdate(match && match[1] ? parseInt(match[1], 10) * 60 : 0);
        } else if (lastCheck.includes("h ago")) {
          const match = lastCheck.match(/(\d+)h ago/);
          setSecondsSinceUpdate(match && match[1] ? parseInt(match[1], 10) * 3600 : 0);
        } else {
          setSecondsSinceUpdate(0);
        }
      }
    };
    
    calculateSeconds();
    const interval = setInterval(calculateSeconds, 1000);
    
    return () => clearInterval(interval);
  }, [monitor.updatedAt, monitor.lastCheck]);

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -20 }}
        transition={{ duration: 0.4 }}
        className="space-y-6"
      >
        <div className={`bg-[#12151a] border p-8 ${
          isPaused ? "border-[#94a3b8]/50" : "border-[#22262f]"
        }`}>
          {isPaused && (
            <div className="mb-4 p-3 bg-[#94a3b8]/10 border border-[#94a3b8]/30 flex items-center gap-2">
              <Pause className="h-4 w-4 text-[#94a3b8]" />
              <span className="text-sm font-semibold text-[#94a3b8]">Monitoring is paused</span>
            </div>
          )}
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-4">
              {monitor.icon ? (
                <div className="p-4 bg-[#0a0c10] border border-[#22262f] text-4xl">
                  {monitor.icon}
                </div>
              ) : (
                <div className="p-4 bg-[#0a0c10] border border-[#22262f]">
                  <Globe className="h-8 w-8 text-[#94a3b8]" />
                </div>
              )}
              <div>
                <h2 className="text-3xl font-bold tracking-tight text-[#e2e8f0] mb-1">
                  {monitor.name}
                </h2>
                <p className="text-[#94a3b8] mb-2 font-mono text-sm">{monitor.url}</p>
                <div className="flex items-center gap-4 text-xs text-[#94a3b8] font-mono">
                  <div className="flex items-center gap-1.5">
                    <Clock className="h-3.5 w-3.5" />
                    <span>Last update: {secondsSinceUpdate === 0 ? "just now" : formatTimeAgo(secondsSinceUpdate)}</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <RefreshCw className="h-3.5 w-3.5" />
                    <span>Interval: {monitor.checkInterval || 60}s</span>
                  </div>
                </div>
              </div>
            </div>
            <div className="flex gap-3">
              <Button 
                variant="outline" 
                size="sm" 
                className={`rounded-none border-[#22262f] bg-transparent text-[#e2e8f0] hover:bg-[#22262f] hover:text-[#e2e8f0] font-mono text-xs uppercase tracking-wider ${
                  isPaused ? "border-[#94a3b8] text-[#94a3b8] hover:bg-[#94a3b8]/10" : ""
                }`}
                onClick={() => onTogglePause(monitor.id, !isPaused)}
              >
                {isPaused ? (
                  <>
                    <Play className="h-4 w-4 mr-2" />
                    Resume
                  </>
                ) : (
                  <>
                    <Pause className="h-4 w-4 mr-2" />
                    Pause
                  </>
                )}
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                className="rounded-none border-[#22262f] bg-transparent text-[#e2e8f0] hover:bg-[#22262f] hover:text-[#e2e8f0] font-mono text-xs uppercase tracking-wider"
                onClick={() => onEdit(monitor)}
              >
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
              <Button 
                variant="destructive" 
                size="sm"
                className="rounded-none bg-[#ff3366] hover:bg-[#ff3366]/80 text-white font-mono text-xs uppercase tracking-wider"
                onClick={() => onDelete(monitor.id)}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            <div className="bg-[#0a0c10] border border-[#22262f] p-4">
              <div className="flex items-center gap-2 mb-2">
                <Zap className="h-4 w-4 text-[#e2e8f0]" />
                <span className="text-xs font-mono tracking-widest text-[#94a3b8] uppercase">Current</span>
              </div>
              <p className="text-2xl font-bold font-mono text-[#00e59b]">
                {monitor.status === "up" ? `${monitor.responseTime}ms` : "N/A"}
              </p>
            </div>
            <div className="bg-[#0a0c10] border border-[#22262f] p-4">
              <div className="flex items-center gap-2 mb-2">
                <TrendingUp className="h-4 w-4 text-[#e2e8f0]" />
                <span className="text-xs font-mono tracking-widest text-[#94a3b8] uppercase">Avg (24h)</span>
              </div>
              <p className="text-2xl font-bold font-mono text-[#e2e8f0]">
                {avgResponseTime > 0 ? `${avgResponseTime}ms` : "N/A"}
              </p>
            </div>
            <div className="bg-[#0a0c10] border border-[#22262f] p-4">
              <div className="flex items-center gap-2 mb-2">
                <Activity className="h-4 w-4 text-[#e2e8f0]" />
                <span className="text-xs font-mono tracking-widest text-[#94a3b8] uppercase">Uptime</span>
              </div>
              <p className="text-2xl font-bold font-mono text-[#e2e8f0]">
                {monitor.uptime ? `${Math.round(monitor.uptime)}%` : "N/A"}
              </p>
            </div>
            <div className="bg-[#0a0c10] border border-[#22262f] p-4">
              <div className="flex items-center gap-2 mb-2">
                <AlertCircle className="h-4 w-4 text-[#e2e8f0]" />
                <span className="text-xs font-mono tracking-widest text-[#94a3b8] uppercase">Status</span>
              </div>
              {isPaused ? (
                <div className="inline-block bg-[#94a3b8]/20 text-[#94a3b8] border-[#94a3b8]/30 border px-3 py-1 font-mono text-xs uppercase tracking-widest">
                  Paused
                </div>
              ) : (
                <div
                  className={`inline-block border px-3 py-1 font-mono text-xs uppercase tracking-widest ${
                    monitor.status === "up"
                      ? "bg-[#00e59b]/10 text-[#00e59b] border-[#00e59b]/30"
                      : "bg-[#ff3366]/10 text-[#ff3366] border-[#ff3366]/30"
                  }`}
                >
                  {monitor.status === "up" ? "Online" : "Offline"}
                </div>
              )}
            </div>
          </div>

          <div className="bg-[#0a0c10] border border-[#22262f] p-6">
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-3">
                <BarChart3 className="h-5 w-5 text-[#94a3b8]" />
                <h3 className="text-lg font-bold text-[#e2e8f0]">Response Time History</h3>
              </div>
              <div className="flex items-center gap-2">
                {(["1h", "12h", "1w", "1y"] as const).map((range) => (
                  <Button
                    key={range}
                    variant={timeRange === range ? "default" : "outline"}
                    size="sm"
                    className={`rounded-none font-mono text-xs px-3 h-7 transition-all ${
                      timeRange === range
                        ? "bg-[#00e59b] text-[#0a0c10] border-[#00e59b] font-bold"
                        : "border-[#22262f] bg-transparent text-[#94a3b8] hover:bg-[#22262f] hover:text-[#e2e8f0]"
                    }`}
                    onClick={() => {
                      setTimeRange(range);
                      if (onFetchResponseTime) {
                        onFetchResponseTime(String(monitor.id), range);
                      }
                    }}
                  >
                    {range === "1h" ? "1H" : range === "12h" ? "12H" : range === "1w" ? "1W" : "1Y"}
                  </Button>
                ))}
              </div>
            </div>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={formattedResponseTimeData}>
                <defs>
                  <linearGradient id="colorGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#00e59b" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#00e59b" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="2 2" stroke="#22262f" opacity={0.5} />
                <XAxis 
                  dataKey="time" 
                  tick={{ fill: "#94a3b8", fontSize: 11, fontFamily: "JetBrains Mono, monospace" }}
                  stroke="#22262f"
                  tickMargin={10}
                />
                <YAxis 
                  tick={{ fill: "#94a3b8", fontSize: 11, fontFamily: "JetBrains Mono, monospace" }}
                  stroke="#22262f"
                  domain={[0, 'auto']}
                  tickMargin={10}
                />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: "#12151a",
                    border: "1px solid #22262f",
                    borderRadius: "0px",
                    color: "#e2e8f0",
                    fontFamily: "JetBrains Mono, monospace",
                    fontSize: "12px"
                  }}
                  itemStyle={{ color: "#00e59b" }}
                  formatter={(value: number | undefined) => [
                    value !== undefined ? `${value.toFixed(2)} ms` : "N/A",
                    "Latency"
                  ]}
                />
                <Area 
                  type="monotone" 
                  dataKey="responseTime" 
                  stroke="#00e59b"
                  strokeWidth={2}
                  fill="url(#colorGradient)"
                  activeDot={{ r: 6, fill: "#00e59b", stroke: "#0a0c10", strokeWidth: 2 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
}

