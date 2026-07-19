import { useState, useEffect, useCallback } from "react";
import { motion } from "framer-motion";
import { Header } from "./components/Header";
import { StatsGrid } from "./components/StatsGrid";
import { ServicesGrid } from "./components/ServicesGrid";
import { MonitorDetails } from "./components/MonitorDetails";
import { AddServiceDialog } from "./components/AddServiceDialog";
import { EditServiceDialog } from "./components/EditServiceDialog";
import type { Monitor, Stats, ResponseTimeData, NewService } from "./types";
import "./index.css";

export function App() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedMonitor, setSelectedMonitor] = useState<Monitor | null>(null);
  const [responseTimeData, setResponseTimeData] = useState<ResponseTimeData[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingMonitor, setEditingMonitor] = useState<Monitor | null>(null);
  const [newService, setNewService] = useState<NewService>({
    name: "",
    url: "",
    isThirdParty: false,
    icon: "",
    checkInterval: 60,
  });
  const [editedService, setEditedService] = useState<NewService>({
    name: "",
    url: "",
    isThirdParty: false,
    icon: "",
    checkInterval: 60,
  });

  const fetchMonitors = useCallback(async () => {
    try {
      const response = await fetch("/api/monitors?t=" + Date.now());
      const data = await response.json();
      setMonitors(data);
      
      if (data.length > 0) {
        setSelectedMonitor((prev) => {
          if (prev) {
            const updatedMonitor = data.find((m: Monitor) => String(m.id) === String(prev.id));
            if (updatedMonitor) {
              return updatedMonitor;
            }
          }
          return data[0];
        });
      }
      setLoading(false);
      setLastUpdate(new Date());
    } catch (error) {
      console.error("Failed to fetch monitors:", error);
      setLoading(false);
    }
  }, []);

  const fetchStats = useCallback(async () => {
    try {
      const response = await fetch("/api/stats?t=" + Date.now());
      const data = await response.json();
      setStats(data);
      setLastUpdate(new Date());
    } catch (error) {
      console.error("Failed to fetch stats:", error);
    }
  }, []);

  const fetchResponseTimeData = useCallback(async (monitorId: string, timeRange: string = "24h") => {
    try {
      const response = await fetch(`/api/response-time?id=${monitorId}&range=${timeRange}&t=${Date.now()}`);
      const data = await response.json();
      setResponseTimeData(data);
      setLastUpdate(new Date());
    } catch (error) {
      console.error("Failed to fetch response time data:", error);
    }
  }, []);

  const createService = async () => {
    try {
      const response = await fetch("/api/monitors/create", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(newService),
      });

      if (!response.ok) {
        const error = await response.text();
        alert("Failed to create service: " + error);
        return;
      }

      setNewService({ name: "", url: "", isThirdParty: false, icon: "", checkInterval: 60 });
      setDialogOpen(false);
      fetchMonitors();
      fetchStats();
    } catch (error) {
      console.error("Failed to create service:", error);
      alert("Failed to create service. Please try again.");
    }
  };

  const deleteService = async (monitorId: string | number) => {
    if (!confirm("Are you sure you want to delete this service?")) {
      return;
    }

    try {
      const response = await fetch(`/api/monitor?id=${monitorId}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        const error = await response.text();
        alert("Failed to delete service: " + error);
        return;
      }

      if (selectedMonitor && String(selectedMonitor.id) === String(monitorId)) {
        setSelectedMonitor(null);
      }

      fetchMonitors();
      fetchStats();
    } catch (error) {
      console.error("Failed to delete service:", error);
      alert("Failed to delete service. Please try again.");
    }
  };

  const handleEdit = (monitor: Monitor) => {
    setEditingMonitor(monitor);
    setEditedService({
      name: monitor.name,
      url: monitor.url,
      isThirdParty: monitor.isThirdParty || false,
      icon: monitor.icon || "",
      checkInterval: monitor.checkInterval || 60,
    });
    setEditDialogOpen(true);
  };

  const updateService = async () => {
    if (!editingMonitor) return;

    try {
      const response = await fetch(`/api/monitor?id=${editingMonitor.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(editedService),
      });

      if (!response.ok) {
        const error = await response.text();
        alert("Failed to update service: " + error);
        return;
      }

      const updatedMonitor = await response.json();
      setEditDialogOpen(false);
      setEditingMonitor(null);
      
      // Update selected monitor if it was the one being edited
      if (selectedMonitor && String(selectedMonitor.id) === String(editingMonitor.id)) {
        setSelectedMonitor(updatedMonitor);
      }

      fetchMonitors();
      fetchStats();
    } catch (error) {
      console.error("Failed to update service:", error);
      alert("Failed to update service. Please try again.");
    }
  };

  const togglePause = async (monitorId: string | number, paused: boolean) => {
    try {
      const response = await fetch(`/api/monitor?id=${monitorId}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ paused }),
      });

      if (!response.ok) {
        const error = await response.text();
        alert("Failed to update pause state: " + error);
        return;
      }

      const updatedMonitor = await response.json();
      
      // Update selected monitor if it was the one being paused/resumed
      if (selectedMonitor && String(selectedMonitor.id) === String(monitorId)) {
        setSelectedMonitor(updatedMonitor);
      }

      fetchMonitors();
      fetchStats();
    } catch (error) {
      console.error("Failed to toggle pause:", error);
      alert("Failed to toggle pause. Please try again.");
    }
  };

  const exportMonitors = async () => {
    try {
      const response = await fetch("/api/monitors/export");
      if (!response.ok) {
        throw new Error("Failed to export monitors");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "monitors.yaml";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Failed to export monitors:", error);
      alert("Failed to export monitors. Please try again.");
    }
  };

  useEffect(() => {
    fetchMonitors();
    fetchStats(); // Initial fetch only
  }, [fetchMonitors, fetchStats]);

  // Set up SSE connection for real-time updates (replaces polling)
  useEffect(() => {
    const eventSource = new EventSource("/api/events");
    
    eventSource.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data);
        
        switch (update.type) {
          case "monitor_update":
            // Update the specific monitor in the list
            setMonitors((prev) => {
              const updated = prev.map((m) => 
                String(m.id) === String(update.data.id) ? update.data : m
              );
              // Update selected monitor if it's the one that changed
              setSelectedMonitor((sel) => {
                if (sel && String(sel.id) === String(update.data.id)) {
                  return update.data;
                }
                return sel;
              });
              return updated;
            });
            setLastUpdate(new Date());
            break;
            
          case "monitor_added":
            // Add new monitor to the list only if it doesn't already exist
            setMonitors((prev) => {
              const exists = prev.some((m) => String(m.id) === String(update.data.id));
              if (exists) {
                // Monitor already exists, update it instead
                return prev.map((m) => 
                  String(m.id) === String(update.data.id) ? update.data : m
                );
              }
              return [...prev, update.data];
            });
            setLastUpdate(new Date());
            break;
            
          case "monitor_deleted":
            // Remove monitor from the list
            setMonitors((prev) => prev.filter((m) => String(m.id) !== String(update.data.id)));
            setSelectedMonitor((sel) => {
              if (sel && String(sel.id) === String(update.data.id)) {
                return null;
              }
              return sel;
            });
            setLastUpdate(new Date());
            break;
            
          case "stats_update":
            // Update stats
            setStats(update.data);
            setLastUpdate(new Date());
            break;
        }
      } catch (error) {
        console.error("Error parsing SSE update:", error);
      }
    };
    
    eventSource.onerror = (error) => {
      console.error("SSE connection error:", error);
      // EventSource will automatically reconnect
    };
    
    return () => {
      eventSource.close();
    };
  }, []);

  useEffect(() => {
    if (selectedMonitor) {
      // Initial fetch only - updates will come via SSE when implemented
      fetchResponseTimeData(String(selectedMonitor.id), "12h");
    }
  }, [selectedMonitor, fetchResponseTimeData]);

  // Removed polling for response time data - will be handled via SSE in future

  const filteredMonitors = monitors.filter(monitor =>
    monitor.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const isHealthy = stats ? stats.servicesDown === 0 : true;

  return (
    <div className="min-h-screen bg-[#0a0c10] text-[#e2e8f0] relative overflow-hidden font-sans">
      {/* Ambient Health Aura */}
      <div 
        className={`absolute top-0 left-0 w-full h-[500px] pointer-events-none transition-colors duration-1000 opacity-20 blur-[120px] ${
          isHealthy ? 'bg-[#00e59b]' : 'bg-[#ff3366]'
        }`}
        style={{ transform: 'translateY(-50%)' }}
      />
      
      <div className="relative z-10 flex flex-col h-screen">
      <Header
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onAddService={() => setDialogOpen(true)}
        onExportMonitors={exportMonitors}
        lastUpdate={lastUpdate}
        isHealthy={isHealthy}
      />

      <div className="flex-1 overflow-hidden px-4 md:px-8 py-6">
        {loading ? (
          <div className="flex items-center justify-center min-h-[60vh]">
            <motion.div
              animate={{ rotate: 360 }}
              transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
              className="w-12 h-12 border-4 border-blue-500/20 border-t-blue-500 rounded-full"
            />
          </div>
        ) : (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.5 }}
            className="space-y-8"
          >
            <StatsGrid stats={stats} />
            {/* Mobile/Tablet: Stack services and details */}
            <div className="lg:hidden space-y-6">
              {selectedMonitor ? (
                <>
                  <ServicesGrid
                    monitors={filteredMonitors}
                    selectedMonitor={selectedMonitor}
                    onSelectMonitor={setSelectedMonitor}
                  />
                  <MonitorDetails
                    monitor={selectedMonitor}
                    responseTimeData={responseTimeData}
                    onDelete={deleteService}
                    onEdit={handleEdit}
                    onTogglePause={togglePause}
                    onFetchResponseTime={fetchResponseTimeData}
                  />
                </>
              ) : (
                <div>
                  <h2 className="text-lg font-bold text-white mb-4">Services</h2>
                  <ServicesGrid
                    monitors={filteredMonitors}
                    selectedMonitor={selectedMonitor}
                    onSelectMonitor={setSelectedMonitor}
                  />
                </div>
              )}
            </div>
            {/* Desktop: Side-by-side layout */}
            <div className="hidden lg:grid lg:grid-cols-4 gap-6">
              <div className="lg:col-span-1">
                <ServicesGrid
                  monitors={filteredMonitors}
                  selectedMonitor={selectedMonitor}
                  onSelectMonitor={setSelectedMonitor}
                />
              </div>
              <div className="lg:col-span-3">
                {selectedMonitor ? (
                  <MonitorDetails
                    monitor={selectedMonitor}
                    responseTimeData={responseTimeData}
                    onDelete={deleteService}
                    onEdit={handleEdit}
                    onTogglePause={togglePause}
                    onFetchResponseTime={fetchResponseTimeData}
                  />
                ) : (
                  <div className="rounded-2xl bg-gradient-to-br from-slate-800/50 to-slate-900/50 backdrop-blur-xl border border-slate-700/50 p-12 shadow-2xl shadow-black/30 flex items-center justify-center min-h-[400px]">
                    <div className="text-center">
                      <p className="text-xl text-slate-400 mb-2">Select a service to view details</p>
                      <p className="text-sm text-slate-500">Choose a service from the list on the left</p>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </motion.div>
        )}
      </div>

      <AddServiceDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        newService={newService}
        onServiceChange={setNewService}
        onCreate={createService}
      />

      <EditServiceDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        monitor={editingMonitor}
        editedService={editedService}
        onServiceChange={setEditedService}
        onUpdate={updateService}
      />
      </div>
    </div>
  );
}

export default App;
