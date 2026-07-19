import { AnimatePresence } from "framer-motion";
import { ServiceCard } from "./ServiceCard";
import type { Monitor } from "../types";

interface ServicesGridProps {
  monitors: Monitor[];
  selectedMonitor: Monitor | null;
  onSelectMonitor: (monitor: Monitor | null) => void;
}

export function ServicesGrid({ monitors, selectedMonitor, onSelectMonitor }: ServicesGridProps) {
  return (
    <div className="bg-[#12151a]">
      {/* Mobile/Tablet: Grid layout (when no selection) */}
      {selectedMonitor === null && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:hidden">
          <AnimatePresence>
            {monitors.map((monitor, index) => {
              return (
                <ServiceCard
                  key={monitor.id}
                  monitor={monitor}
                  isSelected={false}
                  onClick={() => onSelectMonitor(monitor)}
                  index={index}
                />
              );
            })}
          </AnimatePresence>
        </div>
      )}
      {/* Mobile/Tablet: Horizontal scroll layout (when service is selected) */}
      {selectedMonitor !== null && (
        <div className="flex lg:hidden overflow-x-auto pb-2 -mx-4 px-4 snap-x">
          <AnimatePresence>
            {monitors.map((monitor, index) => {
              const isSelected = String(selectedMonitor.id) === String(monitor.id);
              return (
                <div key={monitor.id} className="flex-shrink-0 w-[280px] snap-center">
                  <ServiceCard
                    monitor={monitor}
                    isSelected={isSelected}
                    onClick={() => onSelectMonitor(isSelected ? null : monitor)}
                    index={index}
                  />
                </div>
              );
            })}
          </AnimatePresence>
        </div>
      )}
      {/* Desktop: Vertical list */}
      <div className="hidden lg:block space-y-0">
        <AnimatePresence>
          {monitors.map((monitor, index) => {
            const isSelected = selectedMonitor !== null && String(selectedMonitor.id) === String(monitor.id);
            return (
              <ServiceCard
                key={monitor.id}
                monitor={monitor}
                isSelected={isSelected}
                onClick={() => onSelectMonitor(isSelected ? null : monitor)}
                index={index}
              />
            );
          })}
        </AnimatePresence>
      </div>
    </div>
  );
}

